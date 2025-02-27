package usageprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type inMemoryRecorder struct {
	traces  []ptrace.Traces
	metrics []pmetric.Metrics
	logs    []plog.Logs
}

var _ HoneycombUsageRecorder = (*inMemoryRecorder)(nil)

func (i *inMemoryRecorder) RecordLogsUsage(l plog.Logs) {
	i.logs = append(i.logs, l)
}

func (i *inMemoryRecorder) RecordMetricsUsage(m pmetric.Metrics) {
	i.metrics = append(i.metrics, m)
}

func (i *inMemoryRecorder) RecordTracesUsage(t ptrace.Traces) {
	i.traces = append(i.traces, t)
}

func newInMemoryRecorder() *inMemoryRecorder {
	return &inMemoryRecorder{
		traces:  make([]ptrace.Traces, 0),
		metrics: make([]pmetric.Metrics, 0),
		logs:    make([]plog.Logs, 0),
	}
}

func TestProcessorPassesRequestsToRecorder(t *testing.T) {
	recorder := newInMemoryRecorder()
	processor := &usageProcessor{
		recorder: recorder,
	}

	td := ptrace.NewTraces()
	rt := td.ResourceSpans().AppendEmpty()
	st := rt.ScopeSpans().AppendEmpty()
	s := st.Spans().AppendEmpty()
	s.SetName("test span")

	// pass traces into processor
	_, err := processor.processTraces(context.Background(), td)
	require.NoError(t, err)

	// assert that the traces were passed to the recorder
	require.Equal(t, td, recorder.traces[0])
}

func TestProcessorPassesMetricsToRecorder(t *testing.T) {
	recorder := newInMemoryRecorder()
	processor := &usageProcessor{
		recorder: recorder,
	}

	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetName("test metric")

	// pass metrics into processor
	_, err := processor.processMetrics(context.Background(), md)
	require.NoError(t, err)

	// assert that the metrics were passed to the recorder
	require.Equal(t, md, recorder.metrics[0])
}

func TestProcessorPassesLogsToRecorder(t *testing.T) {
	recorder := newInMemoryRecorder()
	processor := &usageProcessor{
		recorder: recorder,
	}

	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	l := sl.LogRecords().AppendEmpty()
	l.SetEventName("test log")

	// pass logs into processor
	_, err := processor.processLogs(context.Background(), ld)
	require.NoError(t, err)

	// assert that the logs were passed to the recorder
	require.Equal(t, ld, recorder.logs[0])
}

func TestProcessorDoesNotFailWhenRecorderIsNil(t *testing.T) {
	processor := &usageProcessor{
		recorder: nil,
	}

	td := ptrace.NewTraces()
	_, err := processor.processTraces(context.Background(), td)
	require.NoError(t, err)

	md := pmetric.NewMetrics()
	_, err = processor.processMetrics(context.Background(), md)
	require.NoError(t, err)

	ld := plog.NewLogs()
	_, err = processor.processLogs(context.Background(), ld)
	require.NoError(t, err)
}
