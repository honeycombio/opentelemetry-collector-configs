# Creating a collector with the required plugins

Using the configuration provided by this repository requires a build of OpenTelemetry Collector that contains the following plugins:

* [the `hostmetrics` receiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/hostmetricsreceiver#readme): Lives in the contrib repository
* [the `resourcedetection` processor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/resourcedetectionprocessor): Lives in the contrib repository
* [the `metricstransform` processor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/metricstransformprocessor#readme): Lives in the contrib repository
* [the `timestamp` processor](../timestampprocessor): Lives in this repository

A barebones build that contains *only* these plugins is provided in [the "releases" section of this repository](https://github.com/honeycombio/opentelemetry-collector-configs/releases). If you need a different build, you can use [`opentelemetry-collector-builder`](https://github.com/open-telemetry/opentelemetry-collector-builder) to create it - [documentation for this tool can be found here](https://github.com/open-telemetry/opentelemetry-collector-builder#opentelemetry-collector-builder).

Here is a configuration for opentelemetry-collector-builder that will include the `metricstransform` processor and the `timestamp` processor:

```yaml
dist:
  module: github.com/open-telemetry/opentelemetry-collector-builder
  include_core: true
  otelcol_version: "0.30.0"
processors:
  - gomod: "github.com/open-telemetry/opentelemetry-collector-contrib/processor/metricstransformprocessor v0.30.0"
  - gomod: "github.com/honeycombio/opentelemetry-collector-configs/timestampprocessor v0.1.0"
```
