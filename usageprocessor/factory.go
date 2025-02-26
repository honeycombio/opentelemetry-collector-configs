//go:generate mdatagen metadata.yaml

package usageprocessor

import (
	"context"
	"sync"

	"github.com/honeycombio/opentelemetry-collector-configs/usageprocessor/internal/metadata"
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
	processor, err := getOrCreateProcessor(set.ID)
	if err != nil {
		return nil, err
	}
	return processorhelper.NewTraces(
		ctx,
		set,
		cfg,
		nextConsumer,
		processor.processTraces,
		processorhelper.WithStart(processor.Start),
	)
}

func createMetricsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	processor, err := getOrCreateProcessor(set.ID)
	if err != nil {
		return nil, err
	}
	return processorhelper.NewMetrics(
		ctx,
		set,
		cfg,
		nextConsumer,
		processor.processMetrics,
		processorhelper.WithStart(processor.Start),
	)
}

func createLogsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	processor, err := getOrCreateProcessor(set.ID)
	if err != nil {
		return nil, err
	}
	return processorhelper.NewLogs(
		ctx,
		set,
		cfg,
		nextConsumer,
		processor.processLogs,
		processorhelper.WithStart(processor.Start),
	)
}

var processorsMap = map[component.ID]*usageProcessor{}
var processorsMux = sync.Mutex{}

func getOrCreateProcessor(id component.ID) (*usageProcessor, error) {
	processorsMux.Lock()
	defer processorsMux.Unlock()

	if processor, ok := processorsMap[id]; ok {
		return processor, nil
	}

	processor, err := newUsageProcessor()
	if err != nil {
		return nil, err
	}

	processorsMap[id] = processor
	return processor, nil
}
