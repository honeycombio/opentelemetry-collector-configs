package dedupeprocessor

import (
	"time"

	"go.opentelemetry.io/collector/component"
)

type Config struct {
	// TTL is the time-to-live for each entry in the cache. Default is 30 seconds.
	TTL time.Duration `mapstructure:"ttl"`
	// MaxEntries is the maximum number of entries that can be stored in the cache. Default is 1000.
	MaxEntries int `mapstructure:"max_entries"`
}

var _ component.Config = (*Config)(nil)

// Validate checks if the processor configuration is valid
func (cfg *Config) Validate() error {
	// TODO: Add validation logic
	return nil
}
