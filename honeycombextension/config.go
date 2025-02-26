package honeycombextension

import "go.opentelemetry.io/collector/component"

type Config struct {
	OpAMPExtensionID component.ID `mapstructure:"opampextensionID"`
}

func (c *Config) Validate() error {
	return nil
}
