package reduceprocessor

import (
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatautil"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

type cacheKey [16]byte

func newCacheKey(groupBy []string, resource pcommon.Resource, scope pcommon.InstrumentationScope, lr plog.LogRecord) (cacheKey, bool) {
	// create a map to hold group by attributes
	groupByAttrs := pcommon.NewMap()

	// loop over group by attributes and try to find them in log record, scope and resource
	for _, attrName := range groupBy {
		// try to find each attribute in log record, scope and resource
		// done in reverse order so that log record attributes take precedence
		// over scope attributes and scope attributes take precedence over resource attributes
		attr, ok := lr.Attributes().Get(attrName)
		if ok {
			attr.CopyTo(groupByAttrs.PutEmpty(attrName))
			continue
		}
		if attr, ok = scope.Attributes().Get(attrName); ok {
			attr.CopyTo(groupByAttrs.PutEmpty(attrName))
			continue
		}
		if attr, ok = resource.Attributes().Get(attrName); ok {
			attr.CopyTo(groupByAttrs.PutEmpty(attrName))
		}
	}

	var key cacheKey
	if groupByAttrs.Len() == 0 {
		// no group by attributes found so we can't aggregate
		return key, false
	}

	// generate hashes for group by attrs, body and severity
	groupByAttrsHash := pdatautil.MapHash(groupByAttrs)
	bodyHash := pdatautil.ValueHash(lr.Body())
	severityHash := pdatautil.ValueHash(pcommon.NewValueStr(lr.SeverityText()))

	// generate hash for log record
	hash := xxhash.New()
	hash.Write(groupByAttrsHash[:])
	hash.Write(bodyHash[:])
	hash.Write(severityHash[:])

	copy(key[:], hash.Sum(nil))
	return key, true
}

type cacheEntry struct {
	createdAt time.Time
	resource  pcommon.Resource
	scope     pcommon.InstrumentationScope
	log       plog.LogRecord
	count     int
	firstSeen pcommon.Timestamp
	lastSeen  pcommon.Timestamp
}

func newCacheEntry(resource pcommon.Resource, scope pcommon.InstrumentationScope, log plog.LogRecord) *cacheEntry {
	return &cacheEntry{
		createdAt: time.Now().UTC(),
		resource:  resource,
		scope:     scope,
		log:       log,
		firstSeen: log.Timestamp(),
		lastSeen:  log.Timestamp(),
	}
}

func (entry *cacheEntry) merge(mergeStrategies map[string]MergeStrategy, resource pcommon.Resource, scope pcommon.InstrumentationScope, logRecord plog.LogRecord) {
	entry.lastSeen = entry.log.Timestamp()
	mergeAttributes(mergeStrategies, entry.resource.Attributes(), resource.Attributes())
	mergeAttributes(mergeStrategies, entry.scope.Attributes(), scope.Attributes())
	mergeAttributes(mergeStrategies, entry.log.Attributes(), logRecord.Attributes())
}

func (entry *cacheEntry) IncrementCount(mergeCount int) {
	entry.count += mergeCount
}

func mergeAttributes(mergeStrategies map[string]MergeStrategy, existingAttrs pcommon.Map, additionalAttrs pcommon.Map) {
	// loop over new attributes and apply merge strategy
	additionalAttrs.Range(func(attrName string, attrValue pcommon.Value) bool {
		// get merge strategy using attribute name
		mergeStrategy, ok := mergeStrategies[attrName]
		if !ok {
			// use default merge strategy if no strategy is defined for the attribute
			mergeStrategy = First
		}

		switch mergeStrategy {
		case First:
			// add attribute if it doesn't exist
			_, ok := existingAttrs.Get(attrName)
			if !ok {
				attrValue.CopyTo(existingAttrs.PutEmpty(attrName))
			}
		case Last:
			// overwrite existing attribute if present
			attrValue.CopyTo(existingAttrs.PutEmpty(attrName))
		case Array:
			// append value to existing value if it exists
			existingValue, ok := existingAttrs.Get(attrName)
			if ok {
				// if existing value is a slice, append to it
				// otherwise, create a new slice and append both values
				// NOTE: not sure how this will deal with different data types :/
				var slice pcommon.Slice
				if existingValue.Type() == pcommon.ValueTypeSlice {
					slice = existingValue.Slice()
				} else {
					slice = pcommon.NewSlice()
					existingValue.CopyTo(slice.AppendEmpty())
				}
				attrValue.CopyTo(slice.AppendEmpty())

				// update existing attribute with new slice
				slice.CopyTo(existingAttrs.PutEmptySlice(attrName))
			} else {
				// add new attribute as it doesn't exist yet
				attrValue.CopyTo(existingAttrs.PutEmpty(attrName))
			}
		case Concat:
			// concatenate value with existing value if it exists
			existingValue, ok := existingAttrs.Get(attrName)
			if ok {
				// concatenate existing value with new value using configured delimiter
				strValue := strings.Join([]string{existingValue.AsString(), attrValue.AsString()}, ",")
				existingAttrs.PutStr(attrName, strValue)
			} else {
				// add new attribute as it doesn't exist yet
				attrValue.CopyTo(existingAttrs.PutEmpty(attrName))
			}
		}
		return true
	})
}

func (entry *cacheEntry) isInvalid(maxCount int, maxAge time.Duration) bool {
	if entry.count >= maxCount {
		return true
	}
	if maxAge > 0 && time.Since(entry.createdAt) >= maxAge {
		return true
	}
	return false
}

func (entry *cacheEntry) toLogs(config *Config) plog.Logs {
	logs := plog.NewLogs()

	rl := logs.ResourceLogs().AppendEmpty()
	entry.resource.CopyTo(rl.Resource())

	sl := rl.ScopeLogs().AppendEmpty()
	entry.scope.CopyTo(sl.Scope())

	lr := sl.LogRecords().AppendEmpty()
	entry.log.CopyTo(lr)

	// add merge count, first seen and last seen attributes if configured
	if config.ReduceCountAttribute != "" {
		lr.Attributes().PutInt(config.ReduceCountAttribute, int64(entry.count))
	}
	if config.FirstSeenAttribute != "" {
		lr.Attributes().PutStr(config.FirstSeenAttribute, entry.firstSeen.String())
	}
	if config.LastSeenAttribute != "" {
		lr.Attributes().PutStr(config.LastSeenAttribute, entry.lastSeen.String())
	}

	return logs
}
