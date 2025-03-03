//go:generate mdatagen metadata.yaml

package honeycombextension

import (
	"context"

	"github.com/honeycombio/opentelemetry-collector-configs/honeycombextension/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

func NewFactory() extension.Factory {
	return extension.NewFactory(
		metadata.Type,
		createDefaultConfig,
		createExtension,
		metadata.ExtensionStability,
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		OpAMPExtensionID: component.NewID(component.MustNewType("opamp")),
	}
}

func createExtension(_ context.Context, set extension.Settings, cfg component.Config) (extension.Extension, error) {
	oCfg := cfg.(*Config)
	return newHoneycombExtension(oCfg, set)
}
