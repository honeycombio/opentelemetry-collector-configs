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
	require.Equal(t, int64(0), hnyExt.usage[traces].bytes)
	require.Equal(t, int64(0), hnyExt.usage[traces].count)

	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()
	s := ss.Spans().AppendEmpty()
	s.SetName("test")
	s.Attributes().PutStr("foo", "bar")

	// test measure a size
	hnyExt.RecordTracesUsage(td)
	require.Equal(t, int64(tracesMarshaler.TracesSize(td)), hnyExt.usage[traces].bytes)
	require.Equal(t, int64(td.SpanCount()), int64(1))
}

func TestRecordMetricsUsage(t *testing.T) {
	ext, err := newHoneycombExtension(nil, extensiontest.NewNopSettings(metadata.Type))
	require.NoError(t, err)
	hnyExt, ok := ext.(*honeycombExtension)
	require.True(t, ok)

	// test 0s are not recorded
	hnyExt.RecordMetricsUsage(pmetric.NewMetrics())
	require.Equal(t, int64(0), hnyExt.usage[metrics].bytes)
	require.Equal(t, int64(0), hnyExt.usage[metrics].count)

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
	require.Equal(t, int64(metricsMarshaler.MetricsSize(md)), hnyExt.usage[metrics].bytes)
	require.Equal(t, int64(md.MetricCount()), hnyExt.usage[metrics].count)
}

func TestRecordLogsUsage(t *testing.T) {
	ext, err := newHoneycombExtension(nil, extensiontest.NewNopSettings(metadata.Type))
	require.NoError(t, err)
	hnyExt, ok := ext.(*honeycombExtension)
	require.True(t, ok)

	// test 0s are not recorded
	hnyExt.RecordLogsUsage(plog.NewLogs())
	require.Equal(t, int64(0), hnyExt.usage[logs].bytes)
	require.Equal(t, int64(0), hnyExt.usage[logs].count)

	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	l := sl.LogRecords().AppendEmpty()
	l.Body().SetStr("test body")
	l.Attributes().PutStr("foo", "bar")

	// test measure a size
	hnyExt.RecordLogsUsage(ld)
	require.Equal(t, int64(logsMarshaler.LogsSize(ld)), hnyExt.usage[logs].bytes)
	require.Equal(t, int64(ld.LogRecordCount()), hnyExt.usage[logs].count)
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
	dataMap := map[signal]*usage{
		traces:  {bytes: 1, count: 1},
		metrics: {bytes: 2, count: 2},
		logs:    {bytes: 3, count: 3},
	}

	hnyExt.usage = dataMap

	bytes, err = hnyExt.createUsageReport()
	require.NoError(t, err)

	// usage is replaced with an empty map after creating the report
	assert.True(t, hnyExt.usage.isEmpty())

	expectedMetrics := pmetric.NewMetrics()
	rm := expectedMetrics.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetName("bytes_received")
	m.SetEmptySum()
	m.Sum().SetAggregationTemporality(pmetric.AggregationTemporalityDelta)

	d := m.Sum().DataPoints().AppendEmpty()
	d.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	d.Attributes().PutStr("signal", string(traces))
	d.SetIntValue(dataMap[traces].bytes)

	d = m.Sum().DataPoints().AppendEmpty()
	d.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	d.Attributes().PutStr("signal", string(metrics))
	d.SetIntValue(dataMap[metrics].bytes)

	d = m.Sum().DataPoints().AppendEmpty()
	d.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	d.Attributes().PutStr("signal", string(logs))
	d.SetIntValue(dataMap[logs].bytes)

	m = sm.Metrics().AppendEmpty()
	m.SetName("count_received")
	m.SetEmptySum()
	m.Sum().SetAggregationTemporality(pmetric.AggregationTemporalityDelta)

	d = m.Sum().DataPoints().AppendEmpty()
	d.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	d.Attributes().PutStr("signal", string(traces))
	d.SetIntValue(dataMap[traces].count)

	d = m.Sum().DataPoints().AppendEmpty()
	d.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	d.Attributes().PutStr("signal", string(metrics))
	d.SetIntValue(dataMap[metrics].count)

	d = m.Sum().DataPoints().AppendEmpty()
	d.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	d.Attributes().PutStr("signal", string(logs))
	d.SetIntValue(dataMap[logs].count)

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
