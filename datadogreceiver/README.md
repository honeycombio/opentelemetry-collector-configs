# Datadog APM Receiver

## Overview
The Datadog APM Receiver accepts traces in the Datadog Trace Agent Format

###Supported Datadog APIs
- v0.3 (msgpack and json)
- v0.4 (msgpack and json)
- v0.5 (msgpack custom format)

## OTel Builder
To use this receiver, use the [otel collector builder](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder) and reference this receiver.

Save below yaml in a file builder.yml:

```yaml
dist: 
  #radical
  name: custom-collector
  output_path: "./bin"
  otelcol_version: 0.54.0
exporters:
  - import: go.opentelemetry.io/collector/exporter/otlpexporter
    gomod: go.opentelemetry.io/collector v0.54.0
  - import: go.opentelemetry.io/collector/exporter/loggingexporter
    gomod: go.opentelemetry.io/collector v0.54.0
receivers:
  - import: go.opentelemetry.io/collector/receiver/otlpreceiver
    gomod: go.opentelemetry.io/collector v0.54.0
  - gomod: "github.com/honeycombio/opentelemetry-collector-configs/datadogreceiver v1.5.0"
processors:
  - import: go.opentelemetry.io/collector/processor/batchprocessor
    gomod: go.opentelemetry.io/collector v0.54.0
replaces:
```

Building and running the custom collector 

```bash
builder --config ./builder.yml
./bin/custom-collector --config=./path/to/config.yml
```

## Testing locally
In the replaces section of the builder, reference this directory instead  

```yaml
replaces:
  # a list of "replaces" directives that will be part of the resulting go.mod
  - github.com/honeycombio/opentelemetry-collector-configs/datadogreceiver v1.5.0 => /path/to/this/directory

```

## Configuration

Example:

```yaml
receivers:
  datadog:
    endpoint: 0.0.0.0:8126
    read_timeout: 60s
```

### endpoint (Optional)
The address and port on which this receiver listens for traces on

Default: `0.0.0.0:8126`

### read_timeout (Optional)
The read timeout of the HTTP Server

Default: 60s