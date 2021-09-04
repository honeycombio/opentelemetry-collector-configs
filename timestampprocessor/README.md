# Timestamp Processor for OpenTelemetry Collector

Supported pipeline types: metrics

The timestamp processor will round all timestamps in metrics streams to the nearest `<duration>`.

Examples:

```yaml
processors:
  timestamp:
    round_to_nearest: 1s
```

## Publishing a new version of this module

Please use [semantic versioning standards](https://golang.org/doc/modules/version-numbers) when deciding on a new version number.

First, make sure that all changes are committed and pushed to the main branch.

Then:
```bash
go test ./...
git tag timestampprocessor/v0.1.0 # substitute the appropriate version
git push --follow-tags
```

To confirm that the published module is available:
```bash
go list -m github.com/honeycombio/opentelemetry-collector-configs/timestampprocessor@v0.1.0 
```
