package honeycombextension

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/opampcustommessages"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

var (
	unset      component.ID
	marshaller = pmetric.JSONMarshaler{}
)

// TODO: think about the best way to expose this capability to the processors.
//   - Would it be better to make a generic function and the processor passes in options or something
//
// that identify the metrics and/or its attributes?
type HoneycombUsageRecorder interface {
	RecordTracesUsage(ptrace.Traces)
	RecordMetricsUsage(pmetric.Metrics)
	RecordLogsUsage(plog.Logs)
}

type signal string

const (
	// reportUsageMessageType is the message type for the reportUsage custom message sent over opamp.
	reportUsageMessageType = "reportUsage"

	traces  = signal("traces")
	metrics = signal("metrics")
	logs    = signal("logs")
)

func newBytesReceivedMap() map[signal][]int64 {
	return map[signal][]int64{
		traces:  make([]int64, 0),
		metrics: make([]int64, 0),
		logs:    make([]int64, 0),
	}
}

type honeycombExtension struct {
	config *Config
	set    extension.Settings

	bytesReceivedData map[signal][]int64
	bytesReceivedMux  sync.Mutex
	done              chan struct{}

	telemetryHandler opampcustommessages.CustomCapabilityHandler
}

var _ extension.Extension = (*honeycombExtension)(nil)
var _ HoneycombUsageRecorder = (*honeycombExtension)(nil)

func newHoneycombExtension(cfg *Config, set extension.Settings) (extension.Extension, error) {
	return &honeycombExtension{
		config: cfg,
		set:    set,

		bytesReceivedData: newBytesReceivedMap(),
		bytesReceivedMux:  sync.Mutex{},
		done:              make(chan struct{}),

		telemetryHandler: nil,
	}, nil
}

// Start begins the extension's processing.
func (h *honeycombExtension) Start(_ context.Context, host component.Host) error {
	if h.config.opampExtensionID != unset {
		ext, ok := host.GetExtensions()[h.config.opampExtensionID]
		if !ok {
			return fmt.Errorf("extension %q does not exist", h.config.opampExtensionID.String())
		}

		registry, ok := ext.(opampcustommessages.CustomCapabilityRegistry)
		if !ok {
			return fmt.Errorf("extension %q is not a custom message registry", h.config.opampExtensionID.String())
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
	m := ptrace.ProtoMarshaler{}
	size := m.TracesSize(td)

	h.bytesReceivedMux.Lock()
	h.bytesReceivedData[traces] = append(h.bytesReceivedData[traces], int64(size))
	h.bytesReceivedMux.Unlock()
}

func (h *honeycombExtension) RecordMetricsUsage(md pmetric.Metrics) {
	m := pmetric.ProtoMarshaler{}
	size := m.MetricsSize(md)

	h.bytesReceivedMux.Lock()
	h.bytesReceivedData[metrics] = append(h.bytesReceivedData[metrics], int64(size))
	h.bytesReceivedMux.Unlock()
}

func (h *honeycombExtension) RecordLogsUsage(ld plog.Logs) {
	m := plog.ProtoMarshaler{}
	size := m.LogsSize(ld)

	h.bytesReceivedMux.Lock()
	h.bytesReceivedData[logs] = append(h.bytesReceivedData[logs], int64(size))
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
				h.set.Logger.Error("failed to generate payload", zap.Error(err))
				continue
			}

			sendingChan, err := h.telemetryHandler.SendMessage(reportUsageMessageType, data)
			switch {
			case err == nil:
				break
			case errors.Is(err, types.ErrCustomMessagePending):
				<-sendingChan
				continue
			default:
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
			dp.Attributes().PutStr("signal", string(s))
			dp.SetIntValue(v)
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
