//go:generate mdatagen metadata.yaml

package bytecounterprocessor

import (
	"context"
	"sync"

	"github.com/honeycombio/opentelemetry-collector-configs/bytecounterprocessor/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

func NewFactory() processor.Factory {
	return processor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		processor.WithTraces(createTracesProcessor, component.StabilityLevelDevelopment),
		processor.WithMetrics(createMetricsProcessor, component.StabilityLevelDevelopment),
		processor.WithLogs(createLogsProcessor, component.StabilityLevelDevelopment))
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func createTracesProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	processor, err := getOrCreateProcessor(set.ID, set)
	if err != nil {
		return nil, err
	}
	return processorhelper.NewTracesProcessor(
		ctx,
		set,
		cfg,
		nextConsumer,
		processor.processTraces,
	)
}

func createMetricsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	processor, err := getOrCreateProcessor(set.ID, set)
	if err != nil {
		return nil, err
	}
	return processorhelper.NewMetricsProcessor(
		ctx,
		set,
		cfg,
		nextConsumer,
		processor.processMetrics,
	)
}

func createLogsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	processor, err := getOrCreateProcessor(set.ID, set)
	if err != nil {
		return nil, err
	}
	return processorhelper.NewLogsProcessor(
		ctx,
		set,
		cfg,
		nextConsumer,
		processor.processLogs,
	)
}

var processorsMap = map[component.ID]*byteCounterProcessor{}
var processorsMux = sync.Mutex{}

func getOrCreateProcessor(id component.ID, settings processor.Settings) (*byteCounterProcessor, error) {
	processorsMux.Lock()
	defer processorsMux.Unlock()

	if processor, ok := processorsMap[id]; ok {
		return processor, nil
	}

	processor, err := newByteCountProcessor(settings)
	if err != nil {
		return nil, err
	}

	processorsMap[id] = processor
	return processor, nil
}
