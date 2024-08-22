package reduceprocessor

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
)

type MergeStrategy int

const (
	First MergeStrategy = iota
	Last
	Array
	Concat
)

type Config struct {
	// GroupBy is the list of attribute names used to group and aggregate log records. At least one attribute name is required.
	GroupBy []string `mapstructure:"group_by"`

	// MaxReduceTimeout is the maximum amount of time an aggregated log record can be stored in the cache before it should be considered complete. Default is 60s.
	MaxReduceTimeout time.Duration `mapstructure:"max_reduce_timeout"`

	// MaxReduceCount is the maximum number of log records that can be aggregated together. If the maximum is reached, the current aggregated log record is considered complete and a new aggregated log record is created. Default is 100.
	MaxReduceCount int `mapstructure:"max_reduce_count"`

	// CacheSize is the maximum number of entries that can be stored in the cache. Default is 10000.
	CacheSize int `mapstructure:"cache_size"`

	// MergeStrategies is a map of attribute names to a custom merge strategies. If an attribute is not found in the map, the default merge strategy of `First`` is used.
	MergeStrategies map[string]MergeStrategy `mapstructure:"merge_strategies"`

	// ReduceCountAttribute is the attribute name used to store the count of log records on the aggregated log record'. If empty, the count is not stored. Default is "".
	ReduceCountAttribute string `mapstructure:"reduce_count_attribute"`

	// FirstSeenAttribute is the attribute name used to store the timestamp of the first log record in the aggregated log record. If empty, the last seen time is not stored. Default is "".
	FirstSeenAttribute string `mapstructure:"first_seen_attribute"`

	// LastSeenAttribute is attribute name used to store the timestamp of the last log record in the aggregated log record. If empty, the last seen time is not stored. Default is "".
	LastSeenAttribute string `mapstructure:"last_seen_attribute"`
}

var _ component.Config = (*Config)(nil)

// Validate checks if the processor configuration is valid
func (cfg *Config) Validate() error {
	if len(cfg.GroupBy) == 0 {
		return errors.New("group_by must contain at least one attribute name")
	}
	return nil
}
