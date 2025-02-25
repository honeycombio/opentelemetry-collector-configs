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
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// TODO: think about the best way to expose this capability to the processors.
//   - Would it be better to make a generic function and the processor passes in options or something
//
// that identify the metrics and/or its attributes?
type HoneycombUsageRecorder interface {
	RecordTracesBytesReceived(int64)
	RecordMetricsBytesReceived(int64)
	RecordLogsBytesReceived(int64)
}

type signal string

const (
	// reportUsageMessageType is the message type for the reportUsage custom message sent over opamp.
	reportUsageMessageType = "reportUsage"

	traces  = signal("traces")
	metrics = signal("metrics")
	logs    = signal("logs")
)

type honeycombExtension struct {
	config *Config
	set    extension.Settings

	bytesReceivedData map[signal][]int64
	bytesReceivedMux  sync.Mutex

	telemetryHandler opampcustommessages.CustomCapabilityHandler
}

func newHoneycombExtension(cfg *Config, set extension.Settings) (extension.Extension, error) {
	return &honeycombExtension{
		config: cfg,
		set:    set,

		bytesReceivedData: map[signal][]int64{
			traces:  make([]int64, 0),
			metrics: make([]int64, 0),
			logs:    make([]int64, 0),
		},
		bytesReceivedMux: sync.Mutex{},
		telemetryHandler: nil,
	}, nil
}

var (
	unset component.ID
)

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
		// TODO: I am pretty sure this needs deregistered in the Shutdown function
		h.telemetryHandler = handler

		go h.reportUsage()
	}
	return nil
}

// Shutdown ends the extension's processing.
func (h *honeycombExtension) Shutdown(context.Context) error {
	return nil
}

func (h *honeycombExtension) RecordTracesBytesReceived(v int64) {
	h.bytesReceivedMux.Lock()
	h.bytesReceivedData[traces] = append(h.bytesReceivedData[traces], v)
	h.bytesReceivedMux.Unlock()
}

func (h *honeycombExtension) RecordMetricsBytesReceived(v int64) {
	h.bytesReceivedMux.Lock()
	h.bytesReceivedData[metrics] = append(h.bytesReceivedData[metrics], v)
	h.bytesReceivedMux.Unlock()
}

func (h *honeycombExtension) RecordLogsBytesReceived(v int64) {
	h.bytesReceivedMux.Lock()
	h.bytesReceivedData[logs] = append(h.bytesReceivedData[logs], v)
	h.bytesReceivedMux.Unlock()
}

// TODO: this needs to have a clean shutdown
func (h *honeycombExtension) reportUsage() {
	t := time.NewTicker(time.Second * 30)
	defer t.Stop()

	for {
		select {
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

// TODO: add logic to "pop" all datapoints from the map and create the proper message payload.
// https://github.com/honeycombio/refinery/tree/yingrong/refinery_opamp_bytes_received has an example payload.
func (h *honeycombExtension) createUsageReport() ([]byte, error) {
	// get a copy of the data and clear the map
	h.bytesReceivedMux.Lock()
	usage := h.bytesReceivedData
	h.bytesReceivedData = map[signal][]int64{}
	h.bytesReceivedMux.Unlock()

	// create the metrics payload
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	// TODO: add resource attributes?
	sm := rm.ScopeMetrics().AppendEmpty()
	metric := sm.Metrics().AppendEmpty()
	metric.SetName("bytes_received")
	sum := metric.SetEmptySum()
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)

	for signal, dps := range usage {
		for _, v := range dps {
			dp := sum.DataPoints().AppendEmpty()
			dp.Attributes().PutStr("signal", string(signal))
			dp.SetIntValue(v)
		}
	}

	// marshal the metrics into a byte slice
	marshaller := pmetric.JSONMarshaler{}
	data, err := marshaller.MarshalMetrics(metrics)
	if err != nil {
		h.set.Logger.Error("failed to marshal metrics", zap.Error(err))
		return nil, err
	}
	return data, nil
}
