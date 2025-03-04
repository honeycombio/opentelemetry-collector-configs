package honeycombextension

import (
	"errors"
	"testing"
	"time"

	"github.com/honeycombio/opentelemetry-collector-configs/honeycombextension/internal/metadata"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/opampcustommessages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestRecordTracesUsage(t *testing.T) {
	ext, err := newHoneycombExtension(nil, extensiontest.NewNopSettings(metadata.Type))
	require.NoError(t, err)
	hnyExt, ok := ext.(*honeycombExtension)
	require.True(t, ok)

	// test 0s are not recorded
	hnyExt.RecordTracesUsage(ptrace.NewTraces())
	assert.Len(t, hnyExt.bytesReceivedData, 0)

	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()
	s := ss.Spans().AppendEmpty()
	s.SetName("test")
	s.Attributes().PutStr("foo", "bar")

	// test measure a size
	hnyExt.RecordTracesUsage(td)
	require.Len(t, hnyExt.bytesReceivedData[traces], 1)
	assert.Len(t, hnyExt.bytesReceivedData[metrics], 0)
	assert.Len(t, hnyExt.bytesReceivedData[logs], 0)
	assert.Equal(t, int64(tracesMarshaler.TracesSize(td)), hnyExt.bytesReceivedData[traces][0].value)
}

func TestRecordMetricsUsage(t *testing.T) {
	ext, err := newHoneycombExtension(nil, extensiontest.NewNopSettings(metadata.Type))
	require.NoError(t, err)
	hnyExt, ok := ext.(*honeycombExtension)
	require.True(t, ok)

	// test 0s are not recorded
	hnyExt.RecordMetricsUsage(pmetric.NewMetrics())
	assert.Len(t, hnyExt.bytesReceivedData, 0)

	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetEmptySum()
	d := m.Sum().DataPoints().AppendEmpty()
	d.SetIntValue(1)
	d.Attributes().PutStr("foo", "bar")

	// test measure a size
	hnyExt.RecordMetricsUsage(md)
	require.Len(t, hnyExt.bytesReceivedData[metrics], 1)
	assert.Len(t, hnyExt.bytesReceivedData[traces], 0)
	assert.Len(t, hnyExt.bytesReceivedData[logs], 0)
	assert.Equal(t, int64(metricsMarshaler.MetricsSize(md)), hnyExt.bytesReceivedData[metrics][0].value)
}

func TestRecordLogsUsage(t *testing.T) {
	ext, err := newHoneycombExtension(nil, extensiontest.NewNopSettings(metadata.Type))
	require.NoError(t, err)
	hnyExt, ok := ext.(*honeycombExtension)
	require.True(t, ok)

	// test 0s are not recorded
	hnyExt.RecordLogsUsage(plog.NewLogs())
	assert.Len(t, hnyExt.bytesReceivedData, 0)

	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	l := sl.LogRecords().AppendEmpty()
	l.Body().SetStr("test body")
	l.Attributes().PutStr("foo", "bar")

	// test measure a size
	hnyExt.RecordLogsUsage(ld)
	require.Len(t, hnyExt.bytesReceivedData[logs], 1)
	assert.Len(t, hnyExt.bytesReceivedData[traces], 0)
	assert.Len(t, hnyExt.bytesReceivedData[metrics], 0)
	assert.Equal(t, int64(logsMarshaler.LogsSize(ld)), hnyExt.bytesReceivedData[logs][0].value)
}

func Test_createUsageReport(t *testing.T) {
	ext, err := newHoneycombExtension(nil, extensiontest.NewNopSettings(metadata.Type))
	require.NoError(t, err)
	hnyExt, ok := ext.(*honeycombExtension)
	require.True(t, ok)

	// test empty data returns errEmptyUsageData
	bytes, err := hnyExt.createUsageReport()
	assert.ErrorIs(t, err, errEmptyUsageData)
	assert.Empty(t, bytes)

	// test payload is created correctly
	dataMap := map[signal][]datapoint{
		traces: {
			{
				timestamp: time.Now(),
				value:     1,
			},
		},
		metrics: {
			{
				timestamp: time.Now(),
				value:     1,
			},
		},
		logs: {
			{
				timestamp: time.Now(),
				value:     1,
			},
		},
	}

	hnyExt.bytesReceivedData = dataMap

	bytes, err = hnyExt.createUsageReport()
	require.NoError(t, err)
	assert.Empty(t, hnyExt.bytesReceivedData)

	expectedMetrics := pmetric.NewMetrics()
	rm := expectedMetrics.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetName("bytes_received")
	m.SetEmptySum()
	m.Sum().SetAggregationTemporality(pmetric.AggregationTemporalityDelta)

	d := m.Sum().DataPoints().AppendEmpty()
	d.SetTimestamp(pcommon.NewTimestampFromTime(dataMap[traces][0].timestamp))
	d.Attributes().PutStr("signal", string(traces))
	d.SetIntValue(dataMap[traces][0].value)

	d = m.Sum().DataPoints().AppendEmpty()
	d.SetTimestamp(pcommon.NewTimestampFromTime(dataMap[metrics][0].timestamp))
	d.Attributes().PutStr("signal", string(metrics))
	d.SetIntValue(dataMap[metrics][0].value)

	d = m.Sum().DataPoints().AppendEmpty()
	d.SetTimestamp(pcommon.NewTimestampFromTime(dataMap[logs][0].timestamp))
	d.Attributes().PutStr("signal", string(logs))
	d.SetIntValue(dataMap[logs][0].value)

	expectedBytes, err := marshaller.MarshalMetrics(expectedMetrics)
	require.NoError(t, err)
	assert.Equal(t, len(expectedBytes), len(bytes))
}

func Test_sendUsageReport(t *testing.T) {
	testData := []byte("test usage data")

	setupExtWithMockHandler := func(handleFunc func(msgType string, data []byte) (chan struct{}, error)) *honeycombExtension {
		ext, err := newHoneycombExtension(nil, extensiontest.NewNopSettings(metadata.Type))
		require.NoError(t, err)
		hnyExt, ok := ext.(*honeycombExtension)
		require.True(t, ok)

		hnyExt.telemetryHandler = &mockHandler{handleFunc: handleFunc}

		return hnyExt
	}

	t.Run("successful send", func(t *testing.T) {
		hnyExt := setupExtWithMockHandler(func(msgType string, data []byte) (chan struct{}, error) {
			assert.Equal(t, reportUsageMessageType, msgType)
			assert.Equal(t, testData, data)
			return nil, nil
		})

		shouldRetry := hnyExt.sendUsageReport(testData)
		assert.False(t, shouldRetry, "should not retry on success")
	})

	t.Run("pending message", func(t *testing.T) {
		// Create a channel that we'll close immediately to simulate fast completion
		doneChan := make(chan struct{})
		close(doneChan)

		hnyExt := setupExtWithMockHandler(func(msgType string, data []byte) (chan struct{}, error) {
			return doneChan, types.ErrCustomMessagePending
		})

		result := hnyExt.sendUsageReport(testData)
		assert.True(t, result, "should retry if the last pending message completes")
	})

	t.Run("error sending message", func(t *testing.T) {
		testErr := errors.New("test error")
		hnyExt := setupExtWithMockHandler(func(msgType string, data []byte) (chan struct{}, error) {
			return nil, testErr
		})

		result := hnyExt.sendUsageReport(testData)
		assert.False(t, result, "should not retry if there is an unrecoverable error")
	})
}

var _ opampcustommessages.CustomCapabilityHandler = (*mockHandler)(nil)

type mockHandler struct {
	handleFunc func(msgType string, data []byte) (chan struct{}, error)
}

func (m *mockHandler) Message() <-chan *protobufs.CustomMessage {
	return nil
}

func (m *mockHandler) SendMessage(msgType string, data []byte) (chan struct{}, error) {
	return m.handleFunc(msgType, data)
}

func (m *mockHandler) Unregister() {}
