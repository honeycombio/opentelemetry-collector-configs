package timestampprocessor

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/model/pdata"
)

type filterMetricProcessor struct {
	cfg    *Config
	logger *zap.Logger
}

func newTimestampMetricProcessor(logger *zap.Logger, cfg *Config) (*filterMetricProcessor, error) {
	logger.Info("Metric timestamp processor configured")
	return &filterMetricProcessor{cfg: cfg, logger: logger}, nil
}

// processMetrics filters the given metrics based off the filterMetricProcessor's filters.
func (fmp *filterMetricProcessor) processMetrics(_ context.Context, pdm pdata.Metrics) (pdata.Metrics, error) {
	// TODO code goes here
	return pdm, nil
}
