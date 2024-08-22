package dynamicsamplingprocessor

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
)

type SamplerType string

const (
	EMADynamicSampler    string = "EMADynamicSampler"
	EMAThroughputSampler string = "EMAThroughputSampler"
)

type Config struct {
	Sampler   string   `mapstructure:"sampler"`
	KeyFields []string `mapstructure:"key_fields"`

	// EMADynamicSampler specific configuration
	GoalSampleRate int `mapstructure:"goal_sample_rate"`

	// EMAThroughputSampler specific configuration
	GoalThroughputPerSecond int `mapstructure:"goal_throughput_per_second"`
}

var _ component.Config = (*Config)(nil)

func (cfg *Config) Validate() error {
	if cfg.Sampler == "" {
		return fmt.Errorf("sampler must be set. Valid options: %s, %s", EMADynamicSampler, EMAThroughputSampler)
	}

	if cfg.Sampler != EMADynamicSampler && cfg.Sampler != EMAThroughputSampler {
		return fmt.Errorf("sampler must be set to one of the following: %s, %s", EMADynamicSampler, EMAThroughputSampler)
	}

	if len(cfg.KeyFields) == 0 {
		return fmt.Errorf("Must set at least one attribute to use as a key for dynamic sampling")
	}

	if cfg.Sampler == EMADynamicSampler && cfg.GoalSampleRate <= 0 {
		return fmt.Errorf("EMADynamicSampler goal_sample_rate must be set and greater than 0")
	}

	if cfg.Sampler == EMAThroughputSampler && cfg.GoalThroughputPerSecond <= 0 {
		return fmt.Errorf("EMAThroughputSampler goal_throughput_per_second must be set and greater than 0")
	}

	return nil
}
