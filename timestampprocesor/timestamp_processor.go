package timestampprocessor

import (
	"context"
	"fmt"
	"time"

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

// timestamp snapping comment
func (fmp *filterMetricProcessor) processMetrics(_ context.Context, src pdata.Metrics) (pdata.Metrics, error) {
	// set the timestamps to the nearest second
	for i := 0; i < src.ResourceMetrics().Len(); i++ {
		rm := src.ResourceMetrics().At(i)
		for j := 0; j < rm.InstrumentationLibraryMetrics().Len(); j++ {
			ms := rm.InstrumentationLibraryMetrics().At(j)
			for k := 0; k < ms.Metrics().Len(); k++ {
				m := ms.Metrics().At(k)
				switch m.DataType() {
				case pdata.MetricDataTypeIntGauge:
					dataPoints := m.IntGauge().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						snappedTimestamp := gotDataPoint.Timestamp().AsTime().Truncate(time.Second)
						gotDataPoint.SetTimestamp(pdata.TimestampFromTime(snappedTimestamp))
					}
				case pdata.MetricDataTypeGauge:
					dataPoints := m.Gauge().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						snappedTimestamp := gotDataPoint.Timestamp().AsTime().Truncate(time.Second)
						gotDataPoint.SetTimestamp(pdata.TimestampFromTime(snappedTimestamp))
					}
				case pdata.MetricDataTypeIntSum:
					dataPoints := m.IntSum().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						snappedTimestamp := gotDataPoint.Timestamp().AsTime().Truncate(time.Second)
						gotDataPoint.SetTimestamp(pdata.TimestampFromTime(snappedTimestamp))
					}
				case pdata.MetricDataTypeSum:
					dataPoints := m.Sum().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						snappedTimestamp := gotDataPoint.Timestamp().AsTime().Truncate(time.Second)
						gotDataPoint.SetTimestamp(pdata.TimestampFromTime(snappedTimestamp))
					}
				case pdata.MetricDataTypeHistogram:
					dataPoints := m.Histogram().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						snappedTimestamp := gotDataPoint.Timestamp().AsTime().Truncate(time.Second)
						gotDataPoint.SetTimestamp(pdata.TimestampFromTime(snappedTimestamp))
					}
				case pdata.MetricDataTypeSummary:
					dataPoints := m.Summary().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						snappedTimestamp := gotDataPoint.Timestamp().AsTime().Truncate(time.Second)
						gotDataPoint.SetTimestamp(pdata.TimestampFromTime(snappedTimestamp))
					}
				default:
					fmt.Printf("Unknown type")
					return src, fmt.Errorf("unknwon type: %s", m.DataType().String())
				}
			}
		}
	}
	return src, nil
}

// get the minimum timestamp from the set snapped to the nearest second
func minRoundedTimestamp(timestamps []pdata.Timestamp, roundToNearest time.Duration) pdata.Timestamp {
	var minTs pdata.Timestamp
	var minNanoSeconds int64

	if len(timestamps) > 0 {
		minTs = timestamps[0]
		minNanoSeconds = minTs.AsTime().UnixNano()
	}

	for i := 1; i < len(timestamps); i++ {
		if timestamps[i].AsTime().UnixNano() < minNanoSeconds {
			minTs = timestamps[i]
			minNanoSeconds = minTs.AsTime().UnixNano()
		}
	}

	// round minTs to the nearest second
	// duration := minTs.AsTime().Sub(Time()) * time.Duration // time.Time
	fmt.Printf("mintime: %d\n", minTs.AsTime().UnixNano())
	roundTime := minTs.AsTime().Round(1 * roundToNearest)
	fmt.Printf("rounded mintime: %d\n", roundTime.UnixNano())
	return pdata.TimestampFromTime(roundTime)
}
