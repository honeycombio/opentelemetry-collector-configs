package usageprocessor

import "go.opentelemetry.io/collector/component"

type Config struct {
	HoneycombExtensionID component.ID `mapstructure:"honeycombextensionID"`
}

func (c *Config) Validate() error {
	return nil
}
