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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
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

	meter = otel.GetMeterProvider().Meter("honeycombextension")
)

type signal string

const (
	// reportUsageMessageType is the message type for the reportUsage custom message sent over opamp.
	reportUsageMessageType = "reportUsage"

	traces  = signal("traces")
	metrics = signal("metrics")
	logs    = signal("logs")
)

type bytesReceivedMap map[signal][]datapoint

func newBytesReceivedMap() bytesReceivedMap {
	return make(map[signal][]datapoint)
}

type datapoint struct {
	timestamp time.Time
	value     int64
}

type honeycombExtension struct {
	config *Config
	set    extension.Settings

	bytesReceivedData bytesReceivedMap
	bytesReceivedMux  sync.Mutex
	done              chan struct{}

	telemetryHandler opampcustommessages.CustomCapabilityHandler

	// Instrumentation metrics
	bytesReceivedCounterTraces  metric.Int64Counter
	bytesReceivedCounterMetrics metric.Int64Counter
	bytesReceivedCounterLogs    metric.Int64Counter
	usageReportCounterSuccess   metric.Int64Counter
	usageReportCounterFailure   metric.Int64Counter
}

var _ extension.Extension = (*honeycombExtension)(nil)
var _ usageprocessor.HoneycombUsageRecorder = (*honeycombExtension)(nil)

func newHoneycombExtension(cfg *Config, set extension.Settings) (extension.Extension, error) {
	ext := &honeycombExtension{
		config: cfg,
		set:    set,

		bytesReceivedData: newBytesReceivedMap(),
		bytesReceivedMux:  sync.Mutex{},
		done:              make(chan struct{}),

		telemetryHandler: nil,
	}

	if err := ext.initializeInternalMetrics(); err != nil {
		return nil, err
	}

	return ext, nil
}

func (h *honeycombExtension) initializeInternalMetrics() error {
	var err error
	h.bytesReceivedCounterTraces, err = meter.Int64Counter(
		"honeycomb.usage.bytes_received_traces",
		metric.WithDescription("Total bytes received from traces signal type"),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		return fmt.Errorf("failed to create bytes received counter for traces: %w", err)
	}

	h.bytesReceivedCounterMetrics, err = meter.Int64Counter(
		"honeycomb.usage.bytes_received_metrics",
		metric.WithDescription("Total bytes received from metrics signal type"),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		return fmt.Errorf("failed to create bytes received counter for metrics: %w", err)
	}
	h.bytesReceivedCounterLogs, err = meter.Int64Counter(
		"honeycomb.usage.bytes_received_logs",
		metric.WithDescription("Total bytes received from logs signal type"),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		return fmt.Errorf("failed to create bytes received counter for logs: %w", err)
	}

	h.usageReportCounterSuccess, err = meter.Int64Counter(
		"honeycomb.usage.reports.success",
		metric.WithDescription("Number of usage reports successfully sent"),
	)
	if err != nil {
		return fmt.Errorf("failed to create usage report success counter: %w", err)
	}

	h.usageReportCounterFailure, err = meter.Int64Counter(
		"honeycomb.usage.reports.failure",
		metric.WithDescription("Number of usage reports failed to send"),
	)
	if err != nil {
		return fmt.Errorf("failed to create usage report failure counter: %w", err)
	}
	return nil
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

	attrs := attribute.String("signal", string(traces))
	h.bytesReceivedCounterTraces.Add(context.Background(), int64(size), metric.WithAttributes(attrs))

	h.bytesReceivedMux.Lock()
	h.bytesReceivedData[traces] = append(h.bytesReceivedData[traces], datapoint{timestamp: time.Now(), value: int64(size)})
	h.bytesReceivedMux.Unlock()
}

func (h *honeycombExtension) RecordMetricsUsage(md pmetric.Metrics) {
	size := metricsMarshaler.MetricsSize(md)
	if size == 0 {
		return
	}

	attrs := attribute.String("signal", string(metrics))
	h.bytesReceivedCounterMetrics.Add(context.Background(), int64(size), metric.WithAttributes(attrs))

	h.bytesReceivedMux.Lock()
	h.bytesReceivedData[metrics] = append(h.bytesReceivedData[metrics], datapoint{timestamp: time.Now(), value: int64(size)})
	h.bytesReceivedMux.Unlock()
}

func (h *honeycombExtension) RecordLogsUsage(ld plog.Logs) {
	size := logsMarshaler.LogsSize(ld)
	if size == 0 {
		return
	}

	attrs := attribute.String("signal", string(logs))
	h.bytesReceivedCounterLogs.Add(context.Background(), int64(size), metric.WithAttributes(attrs))

	h.bytesReceivedMux.Lock()
	h.bytesReceivedData[logs] = append(h.bytesReceivedData[logs], datapoint{timestamp: time.Now(), value: int64(size)})
	h.bytesReceivedMux.Unlock()
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

			sendingChan, err := h.telemetryHandler.SendMessage(reportUsageMessageType, data)
			switch {
			case err == nil:
				// Count successful report sent
				h.usageReportCounterSuccess.Add(context.Background(), 1)
			case errors.Is(err, types.ErrCustomMessagePending):
				h.usageReportCounterFailure.Add(context.Background(), 1)
				<-sendingChan
				continue
			default:
				h.usageReportCounterFailure.Add(context.Background(), 1)
				h.set.Logger.Error("failed to send message", zap.Error(err))
			}
		}
	}
}

func (h *honeycombExtension) createUsageReport() ([]byte, error) {
	// get a copy of the data and clear the map
	h.bytesReceivedMux.Lock()
	usage := h.bytesReceivedData
	h.bytesReceivedData = newBytesReceivedMap()
	h.bytesReceivedMux.Unlock()

	if len(usage) == 0 {
		return nil, errEmptyUsageData
	}

	// create the metrics payload
	m := pmetric.NewMetrics()
	rm := m.ResourceMetrics().AppendEmpty()
	// TODO: add resource attributes?
	sm := rm.ScopeMetrics().AppendEmpty()
	metric := sm.Metrics().AppendEmpty()
	metric.SetName("bytes_received")
	sum := metric.SetEmptySum()
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)

	for s, dps := range usage {
		// TODO: Should we do some summing here so our payload is smaller and so papi has to do less?
		for _, v := range dps {
			dp := sum.DataPoints().AppendEmpty()
			dp.SetTimestamp(pcommon.NewTimestampFromTime(v.timestamp))
			dp.Attributes().PutStr("signal", string(s))
			dp.SetIntValue(v.value)
			h.set.Logger.Debug("Adding datapoint", zap.String("signal", string(s)), zap.Int64("value", v.value))
		}
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
