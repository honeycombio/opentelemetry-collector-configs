package timestampprocessor

import (
	"fmt"
	"go.opentelemetry.io/collector/component"
	"time"
)

// Config defines configuration for Resource processor.
type Config struct {
	RoundToNearest *time.Duration `mapstructure:"round_to_nearest"`
}

var _ component.Config = (*Config)(nil)

// Validate checks if the processor configuration is valid
func (cfg *Config) Validate() error {
	if cfg.RoundToNearest == nil {
		return fmt.Errorf("missing required field \"round_to_nearest\"")
	}
	return nil
}
