package reduceprocessor

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor/processortest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/plogtest"
)

func TestProcessLogsDeduplicate(t *testing.T) {
	testCases := []struct {
		name               string
		inputFile          string
		expectedFile       string
		mergeStrategies    map[string]MergeStrategy
		numLogsBeforeFlush int
	}{
		{
			name:         "different record attrs",
			inputFile:    "attrs.yaml",
			expectedFile: "attrs-expected.yaml",
		},
		{
			name:         "merge strategy first",
			inputFile:    "merge.yaml",
			expectedFile: "merge-first-expected.yaml",
			mergeStrategies: map[string]MergeStrategy{
				"location": First,
			},
		},
		{
			name:         "merge strategy last",
			inputFile:    "merge.yaml",
			expectedFile: "merge-last-expected.yaml",
			mergeStrategies: map[string]MergeStrategy{
				"location": Last,
			},
		},
		{
			name:         "merge strategy append",
			inputFile:    "merge.yaml",
			expectedFile: "merge-array-expected.yaml",
			mergeStrategies: map[string]MergeStrategy{
				"location": Array,
			},
		},
		{
			name:         "merge strategy append",
			inputFile:    "merge.yaml",
			expectedFile: "merge-concat-expected.yaml",
			mergeStrategies: map[string]MergeStrategy{
				"location": Concat,
			},
		},
		{
			name:               "skip aggregation when no group by attributes match",
			inputFile:          "skip-aggregation.yaml",
			expectedFile:       "skip-aggregation-expected.yaml",
			numLogsBeforeFlush: 1,
		},
		{
			name:         "duplicate attributes",
			inputFile:    "duplicate-attrs.yaml",
			expectedFile: "duplicate-attrs-expected.yaml",
		},
		{
			name:         "merge counts are preseved",
			inputFile:    "merge-counts.yaml",
			expectedFile: "merge-counts-expected.yaml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			factory := NewFactory()
			cfg := factory.CreateDefaultConfig()
			oCfg := cfg.(*Config)
			oCfg.GroupBy = []string{"partition_id"}
			oCfg.MergeStrategies = tc.mergeStrategies
			oCfg.ReduceCountAttribute = "meta.merge_count"

			sink := new(consumertest.LogsSink)
			p, err := factory.CreateLogsProcessor(context.Background(), processortest.NewNopSettings(), oCfg, sink)
			require.NoError(t, err)

			input, err := golden.ReadLogs(filepath.Join("testdata", tc.inputFile))
			require.NoError(t, err)
			expected, err := golden.ReadLogs(filepath.Join("testdata", tc.expectedFile))
			require.NoError(t, err)

			require.NoError(t, p.ConsumeLogs(context.Background(), input))

			// check aggregated logs are not emitted immediately
			// non-aggregated logs are emitted immediately
			actual := sink.AllLogs()
			require.Len(t, actual, tc.numLogsBeforeFlush)

			// shutdown flushes the cache to evit all entries, causing logs to be emitted
			p.Shutdown(context.Background())
			actual = sink.AllLogs()
			require.Len(t, actual, 1)

			require.NoError(t, plogtest.CompareLogs(expected, actual[0]))
		})
	}
}

func TestMaxMergeCountSendsLogsRecord(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GroupBy = []string{"partition_id"}
	cfg.MaxReduceCount = 1

	sink := new(consumertest.LogsSink)
	p, err := factory.CreateLogsProcessor(context.Background(), processortest.NewNopSettings(), cfg, sink)
	require.NoError(t, err)

	input, err := golden.ReadLogs(filepath.Join("testdata", "max-merge.yaml"))
	require.NoError(t, err)

	require.NoError(t, p.ConsumeLogs(context.Background(), input))

	p.(*reduceProcessor).purgeCache()

	actual := sink.AllLogs()
	require.Len(t, actual, 2)

	for i, actualLog := range actual {
		expectedFileName := fmt.Sprintf("max-merge-expected-%d.yaml", i+1)
		expectedLog, err := golden.ReadLogs(filepath.Join("testdata", expectedFileName))
		require.NoError(t, err)

		require.NoError(t, plogtest.CompareLogs(expectedLog, actualLog))
	}
}

func TestFirstLastSeenAttributes(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GroupBy = []string{"partition_id"}
	cfg.FirstSeenAttribute = "meta.first_seen"
	cfg.LastSeenAttribute = "meta.last_seen"

	sink := new(consumertest.LogsSink)
	p, err := factory.CreateLogsProcessor(context.Background(), processortest.NewNopSettings(), cfg, sink)
	require.NoError(t, err)

	input, err := golden.ReadLogs(filepath.Join("testdata", "first-last-seen.yaml"))
	require.NoError(t, err)

	require.NoError(t, p.ConsumeLogs(context.Background(), input))

	p.(*reduceProcessor).purgeCache()
	actual := sink.AllLogs()
	require.Len(t, actual, 1)

	// remove first_seen and last_seen attributes from actual logs so we can compare the remainder
	require.True(t, actual[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Attributes().Remove("meta.first_seen"))
	require.True(t, actual[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Attributes().Remove("meta.last_seen"))

	expected, err := golden.ReadLogs(filepath.Join("testdata", "first-last-seen-expected.yaml"))
	require.NoError(t, err)

	require.NoError(t, plogtest.CompareLogs(expected, actual[0]))
}

func TestReduceStateShouldEvict(t *testing.T) {
	testCases := []struct {
		name      string
		count     int
		createdAt time.Time
		maxCount  int
		maxAge    time.Duration
		expected  bool
	}{
		{
			name:     "returns true when count is greater than max count",
			count:    5,
			maxCount: 2,
			expected: true,
		},
		{
			name:     "returns true when count is equal to max count",
			count:    2,
			maxCount: 2,
			expected: true,
		},
		{
			name:     "returns false when count is less than max count",
			count:    1,
			maxCount: 2,
			expected: false,
		},
		{
			name:      "returns true when max age is set and state is older than max age",
			count:     1,
			maxCount:  2,
			createdAt: time.Now().Add(-2 * time.Second),
			maxAge:    1 * time.Second,
			expected:  true,
		},
		{
			name:      "returns false when max age is set and state is younger than max age",
			count:     1,
			maxCount:  2,
			createdAt: time.Now(),
			maxAge:    1 * time.Second,
			expected:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			state := cacheEntry{
				count:     tc.count,
				createdAt: tc.createdAt,
			}
			require.Equal(t, tc.expected, state.isInvalid(tc.maxCount, tc.maxAge))
		})
	}
}
