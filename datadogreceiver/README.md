# Datadog APM Receiver

## Overview

The Datadog APM Receiver accepts traces in the Datadog Trace Agent Format.

This module is **Experimental**.

The code was originally written in a [closed PR](https://github.com/open-telemetry/opentelemetry-collector-contrib/pull/5836) from the opentelemetry-collector-contrib repo and then updated to the latest standards.

The plan is to contribute this back into the opentelemetry-collector-contrib repository after it's been used by several users and determined to be useful enough to contribute + maintain in the upstream long-term.

### Supported Datadog APIs
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
  - gomod: "github.com/honeycombio/opentelemetry-collector-configs/datadogreceiver v0.1.0"
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
  - github.com/honeycombio/opentelemetry-collector-configs/datadogreceiver v0.1.0 => /path/to/this/directory

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

## Publishing a new version of this module

Please use [semantic versioning standards](https://golang.org/doc/modules/version-numbers) when deciding on a new version number.

First, make sure that all changes are committed and pushed to the main branch. Make sure version.go is with the correct version number.

Then:
```bash
go test ./...
git tag datadogreceiver/v0.1.0 # substitute the appropriate version
git push --follow-tags
```

To confirm that the published module is available:
```bash
go list -m github.com/honeycombio/opentelemetry-collector-configs/datadogreceiver@v0.1.0 
```
