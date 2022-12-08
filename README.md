[![OSS Lifecycle](https://img.shields.io/osslifecycle/honeycombio/opentelemetry-collector-configs?color=yellow)](https://github.com/honeycombio/home/blob/main/honeycomb-oss-lifecycle-and-practices.md)

👋  Hi there! If you have questions about this repository, please head on over to our Honeycomb Pollenators Slack channel and join us in the [#discuss-metrics channel](https://honeycombpollinators.slack.com/archives/C025CD38GBS) there -- we'll be happy to help you out!

## OpenTelemetry Collector Configuration Generator

Creates a configuration file for OpenTelemetry Collector that:

- Sends OTLP metrics to Honeycomb
- Enables the [hostmetrics receiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/hostmetricsreceiver)
- Transforms metrics from the hostmetrics receiver such that they generate optimally-wide Honeycomb records ([see more about the event transformation here](./docs/metrics-transformation.md))

A current version of the config that this repository generates should be available on the [Releases page](https://github.com/honeycombio/opentelemetry-collector-configs/releases).

In order to use this configuration you will need a version of opentelemetry-collector that contains the `metricstransform` processor and the `timestamp` processor. Binaries for those processors should also be available on the [Releases page](https://github.com/honeycombio/opentelemetry-collector-configs/releases). However, if you would like to build your own binary, [refer to this documentation](./docs/building.md).

## Timestamp processor

This repository contains [code for a `timestamp` processor for OpenTelemetry Collector](./timestampprocessor), which allows rounding timestamps in metrics streams to a configurable value.

## Building the config

If you'd like to build a version of the configuration yourself, clone this repo and run `make config`. You'll need these prerequisites available in your `$PATH`:

- [go](https://golang.org/dl/)
- [jq](https://stedolan.github.io/jq/download/)
- [yq](https://kislyuk.github.io/yq/#installation)
- [ocb](https://github.com/open-telemetry/opentelemetry-collector-releases) built from the `opentelemetry-collector-releases` project

Watch updates and rebuild on changes using [`entr`](http://eradman.com/entrproject/) with `ls | entr make`.
