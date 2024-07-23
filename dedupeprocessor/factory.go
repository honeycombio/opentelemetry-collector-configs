package dedupeprocessor

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"

	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/honeycombio/opentelemetry-collector-configs/dedupeprocessor/internal/metadata"
)

var processorCapabilities = consumer.Capabilities{MutatesData: true}

// Note: This isn't a valid configuration because the processor would do no work.
func createDefaultConfig() component.Config {
	return &Config{
		TTL:        time.Second * 30,
		MaxEntries: 1000,
	}
}

// newDedupeLogProcessor returns a processor that modifies attributes of a
// log record. To construct the attributes processors, the use of the factory
// methods are required in order to validate the inputs.
func newDedupeLogProcessor(set processor.Settings, cfg *Config) (*dedupeProcessor, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}
	cache := expirable.NewLRU[[16]byte, bool](cfg.MaxEntries, nil, cfg.TTL)

	return &dedupeProcessor{
		telemetryBuilder: telemetryBuilder,
		cache:            cache,
	}, nil
}

// NewFactory returns a new factory for the Attributes processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		processor.WithLogs(createLogsProcessor, metadata.LogsStability),
	)
}

func createLogsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	oCfg := cfg.(*Config)
	lp, err := newDedupeLogProcessor(set, oCfg)
	if err != nil {
		return nil, err
	}

	return processorhelper.NewLogsProcessor(
		ctx,
		set,
		cfg,
		nextConsumer,
		lp.processLogs,
		processorhelper.WithCapabilities(processorCapabilities))
}
