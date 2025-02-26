package usageprocessor

import (
	"context"
	"fmt"

	hnyext "github.com/honeycombio/opentelemetry-collector-configs/honeycombextension"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var (
	unset component.ID
)

type usageProcessor struct {
	config   *Config
	recorder hnyext.HoneycombUsageRecorder
}

func newUsageProcessor() (*usageProcessor, error) {
	return &usageProcessor{}, nil
}

func (p *usageProcessor) Start(ctx context.Context, host component.Host) error {
	if p.config.honeycombExtensionID != unset {
		ext := host.GetExtensions()[p.config.honeycombExtensionID]
		if ext == nil {
			return fmt.Errorf("extension %q does not exist", p.config.honeycombExtensionID.String())
		}

		recorder, ok := ext.(hnyext.HoneycombUsageRecorder)
		if !ok {
			return fmt.Errorf("extension %q does not implement HoneycombUsageRecorder", p.config.honeycombExtensionID.String())
		}
		p.recorder = recorder
	}
	return nil
}

func (p *usageProcessor) processTraces(ctx context.Context, tracesData ptrace.Traces) (ptrace.Traces, error) {
	p.recorder.RecordTracesUsage(tracesData)
	return tracesData, nil
}

func (p *usageProcessor) processMetrics(ctx context.Context, metricsData pmetric.Metrics) (pmetric.Metrics, error) {
	p.recorder.RecordMetricsUsage(metricsData)
	return metricsData, nil
}

func (p *usageProcessor) processLogs(ctx context.Context, logsData plog.Logs) (plog.Logs, error) {
	p.recorder.RecordLogsUsage(logsData)
	return logsData, nil
}
