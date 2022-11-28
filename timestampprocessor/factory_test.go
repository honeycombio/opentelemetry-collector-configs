package timestampprocessor

import (
	"context"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/config"
)

func TestType(t *testing.T) {
	factory := NewFactory()
	pType := factory.Type()

	assert.Equal(t, pType, component.Type("timestamp"))
}

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.Equal(t, cfg, &Config{
		ProcessorSettings: config.NewProcessorSettings(component.NewID(typeStr)),
	})
	assert.NoError(t, componenttest.CheckConfigStruct(cfg))
}

func TestCreateMetricsProcessor(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.RoundToNearest = getTimeDuration("1s")

	mp, err := factory.CreateMetricsProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), cfg, consumertest.NewNop())
	assert.NotNil(t, mp)
	assert.NoError(t, err)
}
