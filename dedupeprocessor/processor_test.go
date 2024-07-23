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
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)

	sink := new(consumertest.LogsSink)
	p, err := factory.CreateLogsProcessor(context.Background(), processortest.NewNopSettings(), oCfg, sink)
	require.NoError(t, err)

	input, err := golden.ReadLogs(filepath.Join("testdata", "logs.yaml"))
	require.NoError(t, err)
	expected, err := golden.ReadLogs(filepath.Join("testdata", "logs-expected.yaml"))
	require.NoError(t, err)

	assert.NoError(t, p.ConsumeLogs(context.Background(), input))

	actual := sink.AllLogs()
	require.Len(t, actual, 1)

	assert.NoError(t, plogtest.CompareLogs(expected, actual[0]))
}
