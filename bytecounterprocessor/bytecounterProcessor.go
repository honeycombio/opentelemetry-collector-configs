package bytecounterprocessor

import (
	"context"

	"github.com/honeycombio/opentelemetry-collector-configs/bytecounterprocessor/internal/metadata"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
)

type byteCounterProcessor struct {
	telemetryBuilder *metadata.TelemetryBuilder
}

func newByteCountProcessor(settings processor.Settings) (*byteCounterProcessor, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(settings.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	return &byteCounterProcessor{
		telemetryBuilder: telemetryBuilder,
	}, nil
}

func (p *byteCounterProcessor) processTraces(ctx context.Context, tracesData ptrace.Traces) (ptrace.Traces, error) {
	m := ptrace.ProtoMarshaler{}
	p.telemetryBuilder.TracesBytesCount.Add(ctx, int64(m.TracesSize(tracesData)))
	return tracesData, nil
}

func (p *byteCounterProcessor) processMetrics(ctx context.Context, metricsData pmetric.Metrics) (pmetric.Metrics, error) {
	m := pmetric.ProtoMarshaler{}
	p.telemetryBuilder.MetricsBytesCount.Add(ctx, int64(m.MetricsSize(metricsData)))
	return metricsData, nil
}

func (p *byteCounterProcessor) processLogs(ctx context.Context, logsData plog.Logs) (plog.Logs, error) {
	m := plog.ProtoMarshaler{}
	p.telemetryBuilder.LogsBytesCount.Add(ctx, int64(m.LogsSize(logsData)))
	return logsData, nil
}
