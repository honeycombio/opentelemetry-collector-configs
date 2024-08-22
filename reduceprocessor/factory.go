package reduceprocessor

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"

	"github.com/honeycombio/opentelemetry-collector-configs/reduceprocessor/internal/metadata"
)

// NewFactory returns a new factory for the Attributes processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		processor.WithLogs(createLogsProcessor, metadata.LogsStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		GroupBy:              []string{},
		MaxReduceTimeout:     time.Second * 60,
		MaxReduceCount:       100,
		CacheSize:            10_000,
		MergeStrategies:      map[string]MergeStrategy{},
		ReduceCountAttribute: "",
		FirstSeenAttribute:   "",
		LastSeenAttribute:    "",
	}
}

func createLogsProcessor(
	ctx context.Context,
	settings processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	config := cfg.(*Config)
	return newReduceProcessor(ctx, settings, nextConsumer, config)
}
