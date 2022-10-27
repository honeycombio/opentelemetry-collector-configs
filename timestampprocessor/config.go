package timestampprocessor

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/config"
)

// Config defines configuration for Resource processor.
type Config struct {
	config.ProcessorSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct

	RoundToNearest *time.Duration `mapstructure:"round_to_nearest"`
}

var _ config.Processor = (*Config)(nil)

// Validate checks if the processor configuration is valid
func (cfg *Config) Validate() error {
	if cfg.RoundToNearest == nil {
		return fmt.Errorf("missing required field \"round_to_nearest\"")
	}
	return nil
}
