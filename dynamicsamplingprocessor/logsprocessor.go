package dynamicsamplingprocessor

import (
	"context"
	"math/rand"
	"sort"
	"strconv"
	"strings"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.uber.org/zap"

	dynsampler "github.com/honeycombio/dynsampler-go"
)

type logsProcessor struct {
	sampler   dynsampler.Sampler
	keyFields []string

	logger *zap.Logger
}

// newLogsProcessor returns a processor.LogsProcessor that will perform head sampling according to the given
// configuration.
func newLogsProcessor(ctx context.Context, set processor.Settings, nextConsumer consumer.Logs, cfg *Config) (processor.Logs, error) {
	lsp := &logsProcessor{
		sampler:   getSampler(cfg),
		keyFields: cfg.KeyFields,
		logger:    set.Logger,
	}

	return processorhelper.NewLogsProcessor(
		ctx,
		set,
		cfg,
		nextConsumer,
		lsp.processLogs,
		processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}))
}

func getSampler(cfg *Config) dynsampler.Sampler {
	var sampler dynsampler.Sampler
	if cfg.Sampler == EMADynamicSampler {
		sampler = &dynsampler.EMASampleRate{
			GoalSampleRate: cfg.GoalSampleRate,
		}
	} else {
		sampler = &dynsampler.EMAThroughput{
			GoalThroughputPerSec: cfg.GoalThroughputPerSecond,
		}
	}

	sampler.Start()
	return sampler
}

func (lsp *logsProcessor) processLogs(ctx context.Context, logsData plog.Logs) (plog.Logs, error) {
	logsData.ResourceLogs().RemoveIf(func(rl plog.ResourceLogs) bool {
		rl.ScopeLogs().RemoveIf(func(ill plog.ScopeLogs) bool {
			ill.LogRecords().RemoveIf(func(l plog.LogRecord) bool {
				key := makeDynsampleKey(l, lsp.keyFields)
				sampleRate := lsp.sampler.GetSampleRate(key)

				// example: with sampleRate=10, there is a 10% chance rand.Intn(10) == 0
				keep := rand.Intn(sampleRate) == 0
				if keep {
					attrs := l.Attributes()
					attrs.PutInt("SampleRate", int64(sampleRate))
				}

				return !keep
			})
			// Filter out empty ScopeLogs
			return ill.LogRecords().Len() == 0
		})
		// Filter out empty ResourceLogs
		return rl.ScopeLogs().Len() == 0
	})
	if logsData.ResourceLogs().Len() == 0 {
		return logsData, processorhelper.ErrSkipProcessingData
	}
	return logsData, nil
}

func makeDynsampleKey(l plog.LogRecord, keyFields []string) string {
	key := make([]string, len(keyFields))
	var sb strings.Builder

	attrs := l.Attributes()
	for i, field := range keyFields {
		if val, ok := attrs.Get(field); ok {
			switch val.Type() {
			case pcommon.ValueTypeBool:
				key[i] = strconv.FormatBool(val.Bool())
			case pcommon.ValueTypeInt:
				key[i] = strconv.FormatInt(val.Int(), 10)
			case pcommon.ValueTypeDouble:
				key[i] = strconv.FormatFloat(val.Double(), 'E', -1, 64)
			case pcommon.ValueTypeStr:
				key[i] = val.Str()
			default:
				continue
			}
		}
	}

	sort.Strings(key)

	estimatedLength := 0
	for _, k := range key {
		estimatedLength += len(k) + 1 // underscore
	}
	sb.Grow(estimatedLength)

	for i, k := range key {
		if i > 0 {
			sb.WriteByte('_')
		}
		sb.WriteString(k)
	}

	return sb.String()
}
