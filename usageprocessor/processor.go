package usageprocessor

import (
	"context"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
)

type usageProcessor struct {
}

func newUsageProcessor(settings processor.Settings) (*usageProcessor, error) {
	return &usageProcessor{}, nil
}

func (p *usageProcessor) processTraces(ctx context.Context, tracesData ptrace.Traces) (ptrace.Traces, error) {
	// m := ptrace.ProtoMarshaler{}
	// size := m.TracesSize(tracesData)
	return tracesData, nil
}

func (p *usageProcessor) processMetrics(ctx context.Context, metricsData pmetric.Metrics) (pmetric.Metrics, error) {
	// m := pmetric.ProtoMarshaler{}
	// size := m.MetricsSize(metricsData)
	return metricsData, nil
}

func (p *usageProcessor) processLogs(ctx context.Context, logsData plog.Logs) (plog.Logs, error) {
	// m := plog.ProtoMarshaler{}
	// size := m.LogsSize(logsData)
	return logsData, nil
}
