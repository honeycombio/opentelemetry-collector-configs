package usageprocessor

import (
	"context"

	"github.com/honeycombio/opentelemetry-collector-configs/usageprocessor/internal/metadata"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
)

type usageProcessor struct {
	telemetryBuilder *metadata.TelemetryBuilder
}

func newUsageProcessor(settings processor.Settings) (*usageProcessor, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(settings.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	return &usageProcessor{
		telemetryBuilder: telemetryBuilder,
	}, nil
}

func (p *usageProcessor) processTraces(ctx context.Context, tracesData ptrace.Traces) (ptrace.Traces, error) {
	m := ptrace.ProtoMarshaler{}
	p.telemetryBuilder.TracesBytesCount.Add(ctx, int64(m.TracesSize(tracesData)))
	return tracesData, nil
}

func (p *usageProcessor) processMetrics(ctx context.Context, metricsData pmetric.Metrics) (pmetric.Metrics, error) {
	m := pmetric.ProtoMarshaler{}
	p.telemetryBuilder.MetricsBytesCount.Add(ctx, int64(m.MetricsSize(metricsData)))
	return metricsData, nil
}

func (p *usageProcessor) processLogs(ctx context.Context, logsData plog.Logs) (plog.Logs, error) {
	m := plog.ProtoMarshaler{}
	p.telemetryBuilder.LogsBytesCount.Add(ctx, int64(m.LogsSize(logsData)))
	return logsData, nil
}
