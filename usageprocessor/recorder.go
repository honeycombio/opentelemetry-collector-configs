package usageprocessor

import (
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type HoneycombUsageRecorder interface {
	RecordTracesUsage(ptrace.Traces)
	RecordMetricsUsage(pmetric.Metrics)
	RecordLogsUsage(plog.Logs)
}
