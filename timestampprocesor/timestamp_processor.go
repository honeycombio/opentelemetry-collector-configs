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
	// TODO code goes here
	//dataPoints := src.DataPointCount()
	// if dataPoints <= size {
	// 	return src
	// }

	// 	// SetTimestamp replaces the timestamp associated with this IntDataPoint.
	// func (ms IntDataPoint) SetTimestamp(v Timestamp) {
	// 	(*ms.orig).TimeUnixNano = uint64(v)
	// }

	// totalCopiedDataPoints := 0
	// dest := pdata.NewMetrics()

	// outputTimestamps[t] = min(inputTimestamps) + (floor((inputTimestamps[t] - min(inputTimestamps)) / 1sec) * 1sec)

	rms := src.ResourceMetrics()

	// destRs := dest.ResourceMetrics().AppendEmpty()
	// destRs := src.Clone()

	var timestamps []pdata.Timestamp
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		fmt.Printf("rm: %+v\n", rm)

		fmt.Printf("rm.ilm len: %+v\n", rm.InstrumentationLibraryMetrics().Len())
		for j := 0; j < rm.InstrumentationLibraryMetrics().Len(); j++ {

			ms := rm.InstrumentationLibraryMetrics().At(j)
			fmt.Printf("ms: %+v\n", ms)
			fmt.Printf("m len: %+v\n", ms.Metrics().Len())
			for k := 0; k < ms.Metrics().Len(); k++ {
				m := ms.Metrics().At(k)
				fmt.Printf("m: %+v\n", m)
				switch m.DataType() {
				case pdata.MetricDataTypeIntGauge:
					dataPoints := m.IntGauge().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						fmt.Printf("int gauge dp: %+v\n", gotDataPoint)
						timestamps = append(timestamps, gotDataPoint.Timestamp())
					}
				case pdata.MetricDataTypeGauge:
					dataPoints := m.Gauge().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						// ts := gotDataPoint.Timestamp()
						//gotDataPoint.SetTimestamp()
						fmt.Printf("gauge dp ts: %+v\n", gotDataPoint.Timestamp().AsTime().UnixNano())
						timestamps = append(timestamps, gotDataPoint.Timestamp())

					}
				case pdata.MetricDataTypeIntSum:
					dataPoints := m.IntSum().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						fmt.Printf("int sum dp: %+v\n", gotDataPoint)
						timestamps = append(timestamps, gotDataPoint.Timestamp())
					}
				case pdata.MetricDataTypeSum:
					dataPoints := m.Sum().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						fmt.Printf("Sum dp: %+v\n", gotDataPoint)
						timestamps = append(timestamps, gotDataPoint.Timestamp())
					}
				case pdata.MetricDataTypeHistogram:
					dataPoints := m.Histogram().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						fmt.Printf("Histogram dp: %+v\n", gotDataPoint)
						timestamps = append(timestamps, gotDataPoint.Timestamp())
					}
				case pdata.MetricDataTypeSummary:
					dataPoints := m.Summary().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						fmt.Printf("summary dp: %+v\n", gotDataPoint)
						timestamps = append(timestamps, gotDataPoint.Timestamp())
					}
				default:
					fmt.Printf("Unknown type")
				}
			}

		}
	}

	// 1 * time.Second hard code for now to round to nearest second
	minRoundedTs := minRoundedTimestamp(timestamps, 1*time.Second)
	fmt.Printf("len Timestamps: %d\n", len(timestamps))
	fmt.Printf("rounded min timestamp: %d\n", minRoundedTs)

	// seconds := minRoundedTs.AsTime().Unix()
	// fmt.Printf("min seconds: %d\n", seconds)

	// lTS := minRoundedTs.AsTime()
	// newMinTs := pdata.TimestampFromTime(y)

	// fmt.Printf("rounded to nearest seconds: %d\n", newMinTs.AsTime().UnixNano())

	// Now set the timestamps
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)

		for j := 0; j < rm.InstrumentationLibraryMetrics().Len(); j++ {

			ms := rm.InstrumentationLibraryMetrics().At(j)
			for k := 0; k < ms.Metrics().Len(); k++ {
				m := ms.Metrics().At(k)
				switch m.DataType() {
				case pdata.MetricDataTypeIntGauge:
					dataPoints := m.IntGauge().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						fmt.Printf("int gauge dp: %+v\n", gotDataPoint)
						snappedTimestamp := gotDataPoint.Timestamp().AsTime().Truncate(time.Second)
						gotDataPoint.SetTimestamp(pdata.TimestampFromTime(snappedTimestamp))

						// y := gotDataPoint.Timestamp().AsTime()
						// // t := ((math.Floor())/ time.Second * 1) * (time.Second * 1)
						// dur := y.Sub(minRoundedTs.AsTime())
						// fmt.Printf("duration is %+v\n", dur)
						// // math.Floor(dur)
						//						snappedTimestamp := minRoundedTs + (math.Floor((gotDataPoint.Timestamp().AsTime() - minRoundedTs.AsTime()) / 1 * time.Second) * 1 * time.Second)

						//gotDataPoint.SetTimestamp(newMinTs)
					}
				case pdata.MetricDataTypeGauge:
					dataPoints := m.Gauge().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						fmt.Printf("gauge dp ts: %+v\n", gotDataPoint.Timestamp().AsTime().UnixNano())
						snappedTimestamp := gotDataPoint.Timestamp().AsTime().Truncate(time.Second)
						gotDataPoint.SetTimestamp(pdata.TimestampFromTime(snappedTimestamp))
						// // t := ((math.Floor())/ time.Second * 1) * (time.Second * 1)
						// dur := y.Sub(minRoundedTs.AsTime())
						// fmt.Printf("duration is %+v\n", dur)
						// p := math.Floor(dur / time.Second)
						// fmt.Printf("p is %+v\n", p)
						//						snappedTimestamp := minRoundedTs + (math.Floor((gotDataPoint.Timestamp().AsTime() - minRoundedTs.AsTime()) / 1 * time.Second) * 1 * time.Second)

						//gotDataPoint.SetTimestamp(newMinTs)

					}
				case pdata.MetricDataTypeIntSum:
					dataPoints := m.IntSum().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						fmt.Printf("int sum dp: %+v\n", gotDataPoint)
						snappedTimestamp := gotDataPoint.Timestamp().AsTime().Truncate(time.Second)
						gotDataPoint.SetTimestamp(pdata.TimestampFromTime(snappedTimestamp))

						//gotDataPoint.SetTimestamp(newMinTs)
					}
				case pdata.MetricDataTypeSum:
					dataPoints := m.Sum().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						fmt.Printf("Sum dp: %+v\n", gotDataPoint)
						snappedTimestamp := gotDataPoint.Timestamp().AsTime().Truncate(time.Second)
						gotDataPoint.SetTimestamp(pdata.TimestampFromTime(snappedTimestamp))

					}
				case pdata.MetricDataTypeHistogram:
					dataPoints := m.Histogram().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						fmt.Printf("Histogram dp: %+v\n", gotDataPoint)
						snappedTimestamp := gotDataPoint.Timestamp().AsTime().Truncate(time.Second)
						gotDataPoint.SetTimestamp(pdata.TimestampFromTime(snappedTimestamp))

					}
				case pdata.MetricDataTypeSummary:
					dataPoints := m.Summary().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						fmt.Printf("summary dp: %+v\n", gotDataPoint)
						snappedTimestamp := gotDataPoint.Timestamp().AsTime().Truncate(time.Second)
						gotDataPoint.SetTimestamp(pdata.TimestampFromTime(snappedTimestamp))

					}
				default:
					fmt.Printf("Unknown type")
				}
			}

		}
	}

	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		for j := 0; j < rm.InstrumentationLibraryMetrics().Len(); j++ {
			ms := rm.InstrumentationLibraryMetrics().At(j)
			for k := 0; k < ms.Metrics().Len(); k++ {
				m := ms.Metrics().At(k)
				switch m.DataType() {
				case pdata.MetricDataTypeIntGauge:
					dataPoints := m.IntGauge().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						fmt.Printf("int gauge dp ts: %+v\n", gotDataPoint.Timestamp().AsTime().UnixNano())
					}
				case pdata.MetricDataTypeGauge:
					dataPoints := m.Gauge().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						// ts := gotDataPoint.Timestamp()
						//gotDataPoint.SetTimestamp()
						fmt.Printf("gauge dp ts: %+v\n", gotDataPoint.Timestamp().AsTime().UnixNano())

					}
				case pdata.MetricDataTypeIntSum:
					dataPoints := m.IntSum().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						fmt.Printf("int sum dp: %+v\n", gotDataPoint.Timestamp().AsTime().UnixNano())
					}
				case pdata.MetricDataTypeSum:
					dataPoints := m.Sum().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						fmt.Printf("Sum dp: %+v\n", gotDataPoint.Timestamp().AsTime().UnixNano())
					}
				case pdata.MetricDataTypeHistogram:
					dataPoints := m.Histogram().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						fmt.Printf("Histogram dp: %+v\n", gotDataPoint.Timestamp().AsTime().UnixNano())
					}
				case pdata.MetricDataTypeSummary:
					dataPoints := m.Summary().DataPoints()
					for l := 0; l < dataPoints.Len(); l++ {
						gotDataPoint := dataPoints.At(l)
						fmt.Printf("summary dp: %+v\n", gotDataPoint.Timestamp().AsTime().UnixNano())
					}
				default:
					fmt.Printf("Unknown type")
				}
			}

		}
	}
	//fmt.Printf("timestamps: %+v\n", timestamps)

	// srcRs.InstrumentationLibraryMetrics().RemoveIf(func(srcIlm pdata.InstrumentationLibraryMetrics) bool {
	// 	// If we are done skip everything else.
	// 	// if totalCopiedDataPoints == size {
	// 	// 	return false
	// 	// }

	// 	destIlm := destRs.InstrumentationLibraryMetrics().AppendEmpty()
	// 	srcIlm.InstrumentationLibrary().CopyTo(destIlm.InstrumentationLibrary())

	// 	// If possible to move all metrics do that.
	// 	srcDataPointCount := metricSliceDataPointCount(srcIlm.Metrics())
	// 	if size-totalCopiedDataPoints >= srcDataPointCount {
	// 		totalCopiedDataPoints += srcDataPointCount
	// 		srcIlm.Metrics().MoveAndAppendTo(destIlm.Metrics())
	// 		return true
	// 	}

	// 	srcIlm.Metrics().RemoveIf(func(srcMetric pdata.Metric) bool {
	// 		// If we are done skip everything else.
	// 		if totalCopiedDataPoints == size {
	// 			return false
	// 		}
	// 		// If the metric has more data points than free slots we should split it.
	// 		copiedDataPoints, remove := splitMetric(srcMetric, destIlm.Metrics().AppendEmpty(), size-totalCopiedDataPoints)
	// 		totalCopiedDataPoints += copiedDataPoints
	// 		return remove
	// 	})
	// 	return false
	// })
	// return srcRs.InstrumentationLibraryMetrics().Len() == 0

	// return dest

	fmt.Printf("hello world\n")

	fmt.Printf("metrics: %+v\n\n", src.DataPointCount())
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
