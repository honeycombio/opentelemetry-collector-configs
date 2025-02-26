package usageprocessor

import (
	"context"
	"fmt"

	hnyext "github.com/honeycombio/opentelemetry-collector-configs/honeycombextension"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

var (
	unset component.ID
)

type usageProcessor struct {
	logger      *zap.Logger
	extensionID component.ID
	recorder    hnyext.HoneycombUsageRecorder
}

func newUsageProcessor(logger *zap.Logger, cfg *Config) (*usageProcessor, error) {
	return &usageProcessor{
		logger:      logger,
		extensionID: cfg.HoneycombExtensionID,
	}, nil
}

func (p *usageProcessor) Start(ctx context.Context, host component.Host) error {
	if p.extensionID != unset {
		ext := host.GetExtensions()[p.extensionID]
		if ext == nil {
			return fmt.Errorf("extension %q does not exist", p.extensionID.String())
		}

		recorder, ok := ext.(hnyext.HoneycombUsageRecorder)
		if !ok {
			return fmt.Errorf("extension %q does not implement HoneycombUsageRecorder", p.extensionID.String())
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
