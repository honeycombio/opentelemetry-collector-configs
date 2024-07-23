package dedupeprocessor

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor/processortest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/plogtest"
)

func TestProcessLogsDeduplicate(t *testing.T) {
	testCases := []struct {
		name         string
		inputFile    string
		expectedFile string
	}{
		{
			name:         "different record attrs",
			inputFile:    "attrs.yaml",
			expectedFile: "attrs-expected.yaml",
		},
		{
			name:         "different log level",
			inputFile:    "log-level.yaml",
			expectedFile: "log-level-expected.yaml",
		},
		{
			name:         "different log body",
			inputFile:    "log-body.yaml",
			expectedFile: "log-body-expected.yaml",
		},
		{
			name:         "different resource attrs",
			inputFile:    "resource-attrs.yaml",
			expectedFile: "resource-attrs-expected.yaml",
		},
		{
			name:         "different scope attrs",
			inputFile:    "scope-attrs.yaml",
			expectedFile: "scope-attrs-expected.yaml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			factory := NewFactory()
			cfg := factory.CreateDefaultConfig()
			oCfg := cfg.(*Config)

			sink := new(consumertest.LogsSink)
			p, err := factory.CreateLogsProcessor(context.Background(), processortest.NewNopSettings(), oCfg, sink)
			require.NoError(t, err)

			input, err := golden.ReadLogs(filepath.Join("testdata", tc.inputFile))
			require.NoError(t, err)
			expected, err := golden.ReadLogs(filepath.Join("testdata", tc.expectedFile))
			require.NoError(t, err)

			assert.NoError(t, p.ConsumeLogs(context.Background(), input))

			actual := sink.AllLogs()
			require.Len(t, actual, 1)

			assert.NoError(t, plogtest.CompareLogs(expected, actual[0]))
		})
	}
}
