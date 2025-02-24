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
	"go.uber.org/zap"
)

type HoneycombUsageRecorder interface {
	RecordTracesBytesReceived(int64)
	RecordMetricsBytesReceived(int64)
	RecordLogsBytesReceived(int64)
}

type signal string

const (
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
			sendingChan, err := h.telemetryHandler.SendMessage("IDK what this value should be yet", h.generatePayload())
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

// TODO: add logic to "pop" all datapoints from the map and create the proper message payload. Can use refinery as example payload
func (h *honeycombExtension) generatePayload() []byte {
	return nil
}
