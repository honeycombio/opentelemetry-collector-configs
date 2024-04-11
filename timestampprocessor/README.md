

# Deprecation Notice

The Timestamp Processor for OpenTelemetry Collector is deprecated in favor of the [Transform Processor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/transformprocessor). All the functionality of the Timestamp Processor can be achieved via the Transform Processor

```yaml
transform:
  error_mode: ignore
  metric_statements:
    - context: datapoint
      statements:
        - set(time, TruncateTime(time, Duration("1s")))
```

Relevant [OTTL](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl) functions:
  - [set](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/ottlfuncs#set)
  - [TruncateTime](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/ottlfuncs#truncatetime)
  - [Duration](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/ottlfuncs#duration)
