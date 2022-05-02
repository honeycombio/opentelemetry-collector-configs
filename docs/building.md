# Creating a collector with the required plugins

Using the configuration provided by this repository requires a build of OpenTelemetry Collector that contains the following plugins:

* [the `hostmetrics` receiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/hostmetricsreceiver#readme): Lives in the contrib repository
* [the `resourcedetection` processor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/resourcedetectionprocessor): Lives in the contrib repository
* [the `metricstransform` processor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/metricstransformprocessor#readme): Lives in the contrib repository
* [the `healthcheck` extension](ihttps://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/extension/healthcheckextension#readme): Lives in the contrib repository
* [the `timestamp` processor](../timestampprocessor): Lives in this repository

A barebones build that contains *only* these plugins is provided in [the "releases" section of this repository](https://github.com/honeycombio/opentelemetry-collector-configs/releases). If you need a different build, you can use the OpenTelemetry Collector Builder [`ocb`](https://github.com/open-telemetry/opentelemetry-collector/releases) to create it - [documentation for this tool can be found here](https://github.com/open-telemetry/opentelemetry-collector/blob/main/cmd/builder/README.md).

Here is a configuration for the builder that will include the `metricstransform` processor, the `filter` processor, and the `timestamp` processor:

```yaml
dist:
  module: github.com/open-telemetry/opentelemetry-collector-builder
  include_core: true
  otelcol_version: "0.50.0"
processors:
  - gomod: "github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor v0.50.0"
  - gomod: "github.com/open-telemetry/opentelemetry-collector-contrib/processor/metricstransformprocessor v0.50.0"
  - gomod: "github.com/honeycombio/opentelemetry-collector-configs/timestampprocessor v0.4.0"
```
