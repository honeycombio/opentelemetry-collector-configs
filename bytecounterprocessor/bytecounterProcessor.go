package bytecounterprocessor

import (
	"context"

	"github.com/honeycombio/opentelemetry-collector-configs/bytecounterprocessor/internal/metadata"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
)

type bytecounterProcessor struct {
	telemetryBuilder *metadata.TelemetryBuilder
}

func newByteCountProcessor(settings processor.Settings) (*bytecounterProcessor, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(settings.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	return &bytecounterProcessor{
		telemetryBuilder: telemetryBuilder,
	}, nil
}

func (p *bytecounterProcessor) processTraces(ctx context.Context, tracesData ptrace.Traces) (ptrace.Traces, error) {
	m := ptrace.ProtoMarshaler{}
	p.telemetryBuilder.TracesBytesCount.Add(ctx, int64(m.TracesSize(tracesData)))
	return tracesData, nil
}

func (p *bytecounterProcessor) processMetrics(ctx context.Context, metricsData pmetric.Metrics) (pmetric.Metrics, error) {
	m := pmetric.ProtoMarshaler{}
	p.telemetryBuilder.MetricsBytesCount.Add(ctx, int64(m.MetricsSize(metricsData)))
	return metricsData, nil
}

func (p *bytecounterProcessor) processLogs(ctx context.Context, logsData plog.Logs) (plog.Logs, error) {
	m := plog.ProtoMarshaler{}
	p.telemetryBuilder.LogsBytesCount.Add(ctx, int64(m.LogsSize(logsData)))
	return logsData, nil
}
