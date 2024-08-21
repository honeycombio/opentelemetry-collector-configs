package dynamicsamplingprocessor

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		id       string
		expected component.Config
	}{
		{
			name: "EMADynamicSampler correct config",
			id:   "EMADynamicSampler",
			expected: &Config{
				Sampler:        EMADynamicSampler,
				KeyFields:      []string{"key1", "key2"},
				GoalSampleRate: 10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
			require.NoError(t, err)
			processors, err := cm.Sub("processors")
			require.NoError(t, err)

			factory := NewFactory()
			cfg := factory.CreateDefaultConfig()

			sub, err := processors.Sub(tt.id)
			require.NoError(t, err)
			require.NoError(t, sub.Unmarshal(cfg))

			assert.NoError(t, component.ValidateConfig(cfg))
			assert.Equal(t, tt.expected, cfg)
		})
	}
}

// func TestLoadInvalidConfig(t *testing.T) {
// 	for _, test := range []struct {
// 		file     string
// 		contains string
// 	}{
// 		{"invalid_negative.yaml", "sampling rate is negative"},
// 		{"invalid_small.yaml", "sampling rate is too small"},
// 		{"invalid_inf.yaml", "sampling rate is invalid: +Inf%"},
// 		{"invalid_prec.yaml", "sampling precision is too great"},
// 		{"invalid_zero.yaml", "invalid sampling precision"},
// 	} {
// 		t.Run(test.file, func(t *testing.T) {
// 			factories, err := otelcoltest.NopFactories()
// 			require.NoError(t, err)

// 			factory := NewFactory()
// 			factories.Processors[metadata.Type] = factory
// 			// https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/33594
// 			// nolint:staticcheck
// 			_, err = otelcoltest.LoadConfigAndValidate(filepath.Join("testdata", test.file), factories)
// 			require.ErrorContains(t, err, test.contains)
// 		})
// 	}
// }
