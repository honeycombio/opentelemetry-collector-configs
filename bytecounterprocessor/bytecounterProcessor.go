package bytecounterprocessor

import (
	"context"
	"unsafe"

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

func (bcp *bytecounterProcessor) processTraces(ctx context.Context, tracesData ptrace.Traces) (ptrace.Traces, error) {
	byteCount := unsafe.Sizeof(tracesData)
	bcp.telemetryBuilder.TracesBytesCount.Add(ctx, int64(byteCount))
	return tracesData, nil
}

func (bcp *bytecounterProcessor) processMetrics(ctx context.Context, metricsData pmetric.Metrics) (pmetric.Metrics, error) {
	byteCount := unsafe.Sizeof(metricsData)
	bcp.telemetryBuilder.MetricsBytesCount.Add(ctx, int64(byteCount))
	return metricsData, nil
}

func (bcp *bytecounterProcessor) processLogs(ctx context.Context, logsData plog.Logs) (plog.Logs, error) {
	byteCount := unsafe.Sizeof(logsData)
	bcp.telemetryBuilder.LogsBytesCount.Add(ctx, int64(byteCount))
	return logsData, nil
}
