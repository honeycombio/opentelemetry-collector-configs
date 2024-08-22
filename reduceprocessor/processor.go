package reduceprocessor

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/honeycombio/opentelemetry-collector-configs/reduceprocessor/internal/metadata"
)

type reduceProcessor struct {
	telemetryBuilder *metadata.TelemetryBuilder
	nextConsumer     consumer.Logs
	logger           *zap.Logger
	cache            map[cacheKey]*cacheEntry
	config           *Config

	cancel context.CancelFunc
	wg     sync.WaitGroup
	mux    sync.Mutex
}

var _ processor.Logs = (*reduceProcessor)(nil)

func newReduceProcessor(_ context.Context, settings processor.Settings, nextConsumer consumer.Logs, config *Config) (*reduceProcessor, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(settings.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	return &reduceProcessor{
		telemetryBuilder: telemetryBuilder,
		nextConsumer:     nextConsumer,
		logger:           settings.Logger,
		config:           config,
		cache:            make(map[cacheKey]*cacheEntry),
	}, err
}

func (p *reduceProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (p *reduceProcessor) Start(ctx context.Context, _ component.Host) error {
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	p.wg.Add(1)
	go p.handleExportInterval(ctx)

	return nil
}

// handleExportInterval sends metrics at the configured interval.
func (p *reduceProcessor) handleExportInterval(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.MaxReduceTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Export any remaining logs
			p.exportLogs()
			if err := ctx.Err(); err != context.Canceled {
				p.logger.Error("context error", zap.Error(err))
			}
			return
		case <-ticker.C:
			p.exportLogs()
		}
	}
}

// exportLogs exports the logs to the next consumer.
func (p *reduceProcessor) exportLogs() {
	p.mux.Lock()
	defer p.mux.Unlock()

	for k, entry := range p.cache {
		if entry.isInvalid(p.config.MaxReduceCount, p.config.MaxReduceTimeout) {
			p.evictEntry(k, entry)
		}
	}
}

func (p *reduceProcessor) exportLog(entry *cacheEntry) {
	// increment output counter
	p.telemetryBuilder.ReduceProcessorOutput.Add(context.Background(), int64(1))

	// increment number of combined log records
	p.telemetryBuilder.ReduceProcessorCombined.Record(context.Background(), int64(entry.count))

	// create logs from cache entry and send to next consumer
	logs := entry.toLogs(p.config)
	err := p.nextConsumer.ConsumeLogs(context.Background(), logs)
	if err != nil {
		p.logger.Error("Failed to send logs to next consumer", zap.Error(err))
	}
}

func (p *reduceProcessor) Shutdown(_ context.Context) error {
	if p.cancel != nil {
		// Call cancel to stop the export interval goroutine and wait for it to finish.
		p.cancel()
		p.wg.Wait()
	}
	p.purgeCache()
	return nil
}

func (p *reduceProcessor) evictEntry(key cacheKey, entry *cacheEntry) {
	p.exportLog(entry)
	delete(p.cache, key)
}

func (p *reduceProcessor) purgeCache() {
	for k, entry := range p.cache {
		p.evictEntry(k, entry)
	}
}

func (p *reduceProcessor) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	p.mux.Lock()
	defer p.mux.Unlock()

	ld.ResourceLogs().RemoveIf(func(rl plog.ResourceLogs) bool {
		// cache copy of resource attributes
		resource := rl.Resource()

		rl.ScopeLogs().RemoveIf(func(sl plog.ScopeLogs) bool {
			// cache copy of scope attributes
			scope := sl.Scope()

			// increment number of received log records
			p.telemetryBuilder.ReduceProcessorReceived.Add(ctx, int64(sl.LogRecords().Len()))

			sl.LogRecords().RemoveIf(func(logRecord plog.LogRecord) bool {
				// create cache key using resource, scope and log record
				// returns whether we can aggregate the log record or not
				key, canAggregate := newCacheKey(p.config.GroupBy, resource, scope, logRecord)
				if !canAggregate {
					// cannot aggregate, don't remove log record
					return false
				}

				// try to get existing entry from cache
				entry, ok := p.cache[key]
				if !ok {
					// not found, create a new entry
					entry = newCacheEntry(resource, scope, logRecord)
				} else {
					// check if the existing entry is still valid
					if entry.isInvalid(p.config.MaxReduceCount, p.config.MaxReduceTimeout) {
						// not valid, remove it from the cache which triggers onEvict and sends it to the next consumer
						p.evictEntry(key, entry)

						// crete a new entry
						entry = newCacheEntry(resource, scope, logRecord)
					} else {
						// valid, merge log record with existing entry
						entry.merge(p.config.MergeStrategies, resource, scope, logRecord)
					}
				}

				// get merge count from new record, scope or resource attributes and add to the cache entry
				mergeCount := getMergeCount(p.config.ReduceCountAttribute, resource, scope, logRecord)
				entry.IncrementCount(mergeCount)

				// add entry to the cache, replaces existing entry if present
				p.cache[key] = entry

				// remove log record as it has been aggregated
				return true
			})

			// remove if no log records left
			return sl.LogRecords().Len() == 0
		})

		// remove if no scope logs left
		return rl.ScopeLogs().Len() == 0
	})

	// pass any remaining unaggregated log records to the next consumer
	if ld.LogRecordCount() > 0 {
		return p.nextConsumer.ConsumeLogs(ctx, ld)
	}

	return nil
}

// getMergeCount returns the merge count from the log record, scope or resource attributes
// order matters, log record attributes take precedence over scope attributes and scope attributes take precedence over resource attributes
// return 1 if not found
func getMergeCount(name string, resource pcommon.Resource, scope pcommon.InstrumentationScope, logRecord plog.LogRecord) int {
	attr, ok := logRecord.Attributes().Get(name)
	if ok {
		return int(attr.Int())
	}
	if attr, ok = scope.Attributes().Get(name); ok {
		return int(attr.Int())
	}
	if attr, ok = resource.Attributes().Get(name); ok {
		return int(attr.Int())
	}
	return 1
}
