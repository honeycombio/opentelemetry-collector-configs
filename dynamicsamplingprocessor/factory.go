//go:generate mdatagen metadata.yaml

package dynamicsamplingprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
)

func NewFactory() processor.Factory {
	return processor.NewFactory(
		component.MustNewType("dynamic_sampler"),
		createDefaultConfig,
		processor.WithLogs(createLogsProcessor, component.StabilityLevelDevelopment))
}

func createDefaultConfig() component.Config {
	return &Config{
		Sampler:        EMADynamicSampler,
		KeyFields:      []string{"key1", "key2"},
		GoalSampleRate: 10,
	}
}

// createLogsProcessor creates a log processor based on this config.
func createLogsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	return newLogsProcessor(ctx, set, nextConsumer, cfg.(*Config))
}
