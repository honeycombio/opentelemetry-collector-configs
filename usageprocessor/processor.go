package usageprocessor

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

var (
	unset component.ID
)

type usageProcessor struct {
	logger   *zap.Logger
	config   *Config
	recorder HoneycombUsageRecorder
}

func newUsageProcessor(settings processor.Settings, cfg *Config) (*usageProcessor, error) {
	return &usageProcessor{
		logger: settings.Logger,
		config: cfg,
	}, nil
}

func (p *usageProcessor) Start(ctx context.Context, host component.Host) error {
	if p.config.HoneycombExtensionID != unset {
		ext := host.GetExtensions()[p.config.HoneycombExtensionID]
		if ext == nil {
			return fmt.Errorf("extension %q does not exist", p.config.HoneycombExtensionID.String())
		}

		recorder, ok := ext.(HoneycombUsageRecorder)
		if !ok {
			return fmt.Errorf("extension %q does not implement HoneycombUsageRecorder", p.config.HoneycombExtensionID.String())
		}
		p.recorder = recorder
	} else {
		p.logger.Warn("No Honeycomb extension ID set, usage data will not be recorded")
	}
	return nil
}

func (p *usageProcessor) processTraces(ctx context.Context, tracesData ptrace.Traces) (ptrace.Traces, error) {
	if p.recorder != nil {
		p.recorder.RecordTracesUsage(tracesData)
	}
	return tracesData, nil
}

func (p *usageProcessor) processMetrics(ctx context.Context, metricsData pmetric.Metrics) (pmetric.Metrics, error) {
	if p.recorder != nil {
		p.recorder.RecordMetricsUsage(metricsData)
	}
	return metricsData, nil
}

func (p *usageProcessor) processLogs(ctx context.Context, logsData plog.Logs) (plog.Logs, error) {
	if p.recorder != nil {
		p.recorder.RecordLogsUsage(logsData)
	}
	return logsData, nil
}
