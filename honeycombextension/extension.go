package honeycombextension

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/honeycombio/opentelemetry-collector-configs/usageprocessor"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/opampcustommessages"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"

	"github.com/honeycombio/opentelemetry-collector-configs/honeycombextension/internal/metadata"
)

var (
	unset             component.ID
	errEmptyUsageData = errors.New("no usage data to report")

	// JSON marshaler is used to encode the metrics payload that is sent to opamp.
	marshaller = pmetric.JSONMarshaler{}

	// Proto marshalers are used to calculate the size of the signal requests that are recorded.
	tracesMarshaler  = ptrace.ProtoMarshaler{}
	metricsMarshaler = pmetric.ProtoMarshaler{}
	logsMarshaler    = plog.ProtoMarshaler{}
)

const (
	// reportUsageMessageType is the message type for the reportUsage custom message sent over opamp.
	reportUsageMessageType = "reportUsage"

	traces  = signal("traces")
	metrics = signal("metrics")
	logs    = signal("logs")
)

type signal string

type usage struct {
	bytes int64
	count int64
}

type usageMap map[signal]*usage

func (u usageMap) isEmpty() bool {
	return u[traces].bytes == 0 && u[metrics].bytes == 0 && u[logs].bytes == 0
}

func newUsageMap() usageMap {
	return map[signal]*usage{
		traces:  {bytes: 0, count: 0},
		metrics: {bytes: 0, count: 0},
		logs:    {bytes: 0, count: 0},
	}
}

type honeycombExtension struct {
	config *Config
	set    extension.Settings

	usage            usageMap
	bytesReceivedMux sync.Mutex
	done             chan struct{}

	telemetryHandler opampcustommessages.CustomCapabilityHandler

	telemetryBuilder *metadata.TelemetryBuilder
}

var _ extension.Extension = (*honeycombExtension)(nil)
var _ usageprocessor.HoneycombUsageRecorder = (*honeycombExtension)(nil)

func newHoneycombExtension(cfg *Config, set extension.Settings) (extension.Extension, error) {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}
	return &honeycombExtension{
		telemetryBuilder: telemetryBuilder,
		config:           cfg,
		set:              set,

		usage:            newUsageMap(),
		bytesReceivedMux: sync.Mutex{},
		done:             make(chan struct{}),

		telemetryHandler: nil,
	}, nil
}

// Start begins the extension's processing.
func (h *honeycombExtension) Start(_ context.Context, host component.Host) error {
	if h.config.OpAMPExtensionID != unset {
		ext, ok := host.GetExtensions()[h.config.OpAMPExtensionID]
		if !ok {
			return fmt.Errorf("extension %q does not exist", h.config.OpAMPExtensionID.String())
		}

		registry, ok := ext.(opampcustommessages.CustomCapabilityRegistry)
		if !ok {
			return fmt.Errorf("extension %q is not a custom message registry", h.config.OpAMPExtensionID.String())
		}

		handler, err := registry.Register("io.honeycomb.capabilities.sendAgentTelemetry")
		if err != nil {
			return fmt.Errorf("failed to register custom capability: %w", err)
		}
		h.telemetryHandler = handler

		go h.reportUsage()
	}
	return nil
}

// Shutdown ends the extension's processing.
func (h *honeycombExtension) Shutdown(context.Context) error {
	close(h.done)
	if h.telemetryHandler != nil {
		h.telemetryHandler.Unregister()
	}
	return nil
}

func (h *honeycombExtension) RecordTracesUsage(td ptrace.Traces) {
	size := tracesMarshaler.TracesSize(td)
	if size == 0 {
		return
	}

	h.telemetryBuilder.HoneycombExtensionBytesReceivedTraces.Add(context.Background(), int64(size))

	h.bytesReceivedMux.Lock()
	h.usage[traces].bytes += int64(size)
	h.usage[traces].count += int64(td.SpanCount())
	h.bytesReceivedMux.Unlock()
}

func (h *honeycombExtension) RecordMetricsUsage(md pmetric.Metrics) {
	size := metricsMarshaler.MetricsSize(md)
	if size == 0 {
		return
	}

	h.telemetryBuilder.HoneycombExtensionBytesReceivedMetrics.Add(context.Background(), int64(size))

	h.bytesReceivedMux.Lock()
	h.usage[metrics].bytes += int64(size)
	h.usage[metrics].count += int64(md.MetricCount())
	h.bytesReceivedMux.Unlock()
}

func (h *honeycombExtension) RecordLogsUsage(ld plog.Logs) {
	size := logsMarshaler.LogsSize(ld)
	if size == 0 {
		return
	}

	h.telemetryBuilder.HoneycombExtensionBytesReceivedLogs.Add(context.Background(), int64(size))

	h.bytesReceivedMux.Lock()
	h.usage[logs].bytes += int64(size)
	h.usage[logs].count += int64(ld.LogRecordCount())
	h.bytesReceivedMux.Unlock()
}

func (h *honeycombExtension) sendUsageReport(data []byte) (retry bool) {
	sendingChan, err := h.telemetryHandler.SendMessage(reportUsageMessageType, data)

	switch {
	case err == nil:
		h.telemetryBuilder.HoneycombExtensionUsageReportSuccess.Add(context.Background(), 1)
		h.set.Logger.Debug("Successfully sent usage report")
		return false

	case errors.Is(err, types.ErrCustomMessagePending):
		h.telemetryBuilder.HoneycombExtensionUsageReportPending.Add(context.Background(), 1)
		h.set.Logger.Debug("Message pending, waiting for completion")

		<-sendingChan
		return true

	default:
		h.telemetryBuilder.HoneycombExtensionUsageReportFailure.Add(context.Background(), 1)
		h.set.Logger.Error("Failed to send message", zap.Error(err))
		return false
	}
}

func (h *honeycombExtension) reportUsage() {
	t := time.NewTicker(time.Second * 30)
	defer t.Stop()

	for {
		select {
		case <-h.done:
			return
		case <-t.C:
			data, err := h.createUsageReport()
			if err != nil {
				if errors.Is(err, errEmptyUsageData) {
					h.set.Logger.Debug("no usage data to report")
					continue
				}
				h.set.Logger.Error("failed to generate payload", zap.Error(err))
				continue
			}

			shouldRetry := h.sendUsageReport(data)
			// If the message was pending, wait for it to complete and retry once
			if shouldRetry {
				h.set.Logger.Debug("Pending message completed, retrying once")

				failedRetry := h.sendUsageReport(data)
				if failedRetry {
					h.telemetryBuilder.HoneycombExtensionUsageReportFailure.Add(context.Background(), 1)
					h.set.Logger.Error("Failed to send usage report after retry")
				}
			}
		}
	}
}
func (h *honeycombExtension) createUsageReport() ([]byte, error) {
	// get a copy of the data and clear the map
	h.bytesReceivedMux.Lock()
	usage := h.usage
	h.usage = newUsageMap()
	h.bytesReceivedMux.Unlock()

	if usage.isEmpty() {
		return nil, errEmptyUsageData
	}

	// create the metrics payload
	m := pmetric.NewMetrics()
	rm := m.ResourceMetrics().AppendEmpty()
	// TODO: add resource attributes?
	sm := rm.ScopeMetrics().AppendEmpty()
	bytesMetric := sm.Metrics().AppendEmpty()
	bytesMetric.SetName("bytes_received")
	sum := bytesMetric.SetEmptySum()
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)

	for s, v := range usage {
		dp := sum.DataPoints().AppendEmpty()
		dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		dp.Attributes().PutStr("signal", string(s))
		dp.SetIntValue(v.bytes)
		h.set.Logger.Debug("Adding datapoint", zap.String("signal", string(s)), zap.Int64("value", v.bytes))
	}

	countsMetric := sm.Metrics().AppendEmpty()
	countsMetric.SetName("count_received")
	countsSum := countsMetric.SetEmptySum()
	countsSum.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)
	for s, v := range usage {
		dp := countsSum.DataPoints().AppendEmpty()
		dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		dp.Attributes().PutStr("signal", string(s))
		dp.SetIntValue(v.count)
		h.set.Logger.Debug("Adding datapoint", zap.String("signal", string(s)), zap.Int64("value", v.count))
	}

	// marshal the metrics into a byte slice
	// TODO: if this marshal fails, we'll lose all the data we grabbed at the beginning of the function. Should we deal with that?
	data, err := marshaller.MarshalMetrics(m)
	if err != nil {
		h.set.Logger.Error("failed to marshal metrics", zap.Error(err))
		return nil, err
	}
	return data, nil
}
