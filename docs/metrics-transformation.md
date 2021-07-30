# Metrics Transformation for Wider Events in Honeycomb

Honeycomb accepts OTLP metrics and converts them into events, [as described in our documentation](https://docs.honeycomb.io/manage-data-volume/metrics/). To manage costs and improve usability of this data in Honeycomb, we suggest that Honeycomb metrics customers should pack as many metrics datapoints as possible into as few events as possible.

Unfortunately, many common metrics producers create metric data where almost every data point has a unique set of descriptive attributes. This will result in a repackaging ratio of close to 1: in other words, one event per data point. Notably, this includes [OpenTelemetry Collector’s hostmetrics receiver](https://github.com/open-telemetry/opentelemetry-collector/tree/main/receiver/hostmetricsreceiver#readme), which [we strongly encourage you to install](https://docs.honeycomb.io/getting-data-in/metrics/opentelemetry-collector-host-metrics/) since it is the easiest way to obtain standard infrastructure-level metrics. Here’s an example of part of a metrics request generated by this receiver (see the full example (https://gist.github.com/maxedmands/ffaa8aff0bacab742fd0d94422646803)):

```
resource:{
    attributes:{ key:"host.name" value:{ string_value:"vm2" } }
    attributes:{ key:"service.name" value:{ string_value:"webserver" } }
}
metrics:{
  name:"system.memory.usage"
  description:"Bytes of memory in use."
  unit:"By"
  int_sum:{
    data_points:{
      labels:{ key:"state" value:"used" }
      time_unix_nano:1623277287515079370
      value:217608192
    }
    data_points:{
      labels:{ key:"state" value:"free" }
      time_unix_nano:1623277287515079370
      value:79982592
    }
    data_points:{
      labels:{ key:"state" value:"buffered" }
      time_unix_nano:1623277287515079370
      value:28291072
    }
    data_points:{
      labels:{ key:"state" value:"cached" }
      time_unix_nano:1623277287515079370
      value:703111168
    }
    data_points:{
      labels:{ key:"state" value:"slab_reclaimable" }
      time_unix_nano:1623277287515079370
      value:41844736
    }
    data_points:{
      labels:{ key:"state" value:"slab_unreclaimable" }
      time_unix_nano:1623277287515079370
      value:41148416
    }
    aggregation_temporality:AGGREGATION_TEMPORALITY_CUMULATIVE
  }
}
```

Here are the resulting events that are generated from this data. Note that we have received 6 data points, and we have generated 6 events, for a repackaging ratio of 1:

```json
{
  "Timestamp": "2021-06-09T22:21:27.51507937Z",
  "host.name": "vm2",
  "service.name": "webserver",
  "state": "used",
  "system.memory.usage": 217608192
},
{
  "Timestamp": "2021-06-09T22:21:27.51507937Z",
  "host.name": "vm2",
  "service.name": "webserver",
  "state": "slab_unreclaimable",
  "system.memory.usage": 41148416
},
{
  "Timestamp": "2021-06-09T22:21:27.51507937Z",
  "host.name": "vm2",
  "service.name": "webserver",
  "state": "cached",
  "system.memory.usage": 703111168
},
{
  "Timestamp": "2021-06-09T22:21:27.51507937Z",
  "host.name": "vm2",
  "service.name": "webserver",
  "state": "buffered",
  "system.memory.usage": 28291072
},
{
  "Timestamp": "2021-06-09T22:21:27.51507937Z",
  "host.name": "vm2",
  "service.name": "webserver",
  "state": "slab_reclaimable",
  "system.memory.usage": 41844736
},
{
  "Timestamp": "2021-06-09T22:21:27.51507937Z",
  "host.name": "vm2",
  "service.name": "webserver",
  "state": "free",
  "system.memory.usage": 79982592
}
```

## How we repackage metrics for wider events

OpenTelemetry Collector has a modular design that allows users to collect metrics from a variety of sources, process them arbitrarily, and then export them to various endpoints. In particular, a metrics processor will accept a stream of arbitrary metrics, do some operation on them, and then output a resulting stream of metrics. Multiple processors can be set to run in a pipeline, each modifying the metrics stream in a different way.

### Preaggregate away unnecessary labels

The hostmetrics receiver provides certain metrics that are broken out into very granular timeseries, where most users would be very happy just looking at a coarser aggregation of this data. For example, the *system.filesystem.usage* metric is separated into individual timeseries based on:

```
data_points:{
  labels:{ key:"device" value:"/dev/vda1" }
  labels:{ key:"type" value:"ext4" }
  labels:{ key:"mode" value:"rw" }
  labels:{ key:"mountpoint" value:"/" }
  labels:{ key:"state", value: "used" }
  time_unix_nano:1623277287513691424
  value:2008989696
}
data_points:{
  labels:{ key:"device" value:"/dev/loop0" }
  labels:{ key:"type" value:"squashfs" }
  labels:{ key:"mode" value:"ro" }
  labels:{ key:"mountpoint" value:"/snap/core18/2066" }
  labels:{ key:"state", value: "used" }
  time_unix_nano:1623277287513691424
  value:58195968
}
```

The average user will not need to know filesystem usage broken out by device — usage for the host as a whole should be sufficient. The same holds for type, mode, and mountpoint (which incidentally are all additional detail about the device attribute in this case). In this example we should aggregate these datapoints into a single datapoint using a SUM aggregation (i.e., adding the values together). We should retain the state label. The result should look like this:

```
data_points:{
  time_unix_nano:1623277287513691424
  value:2067185664
  labels:{ key:"state", value: "used" }
}
```

 To accomplish this, we should be able to use the *metricstransform* processor (https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/metricstransformprocessor#readme) in the contrib repo for OpenTelemetry collector:

```yaml
processors:
  metricstransform:
    transforms:
      - include: system.filesystem.usage
        action: update
        operations:
          - action: aggregate_labels
            aggregation_type: sum
            label_set: []
```

The hostmetrics reciever comes with a [metadata.yaml file](https://github.com/open-telemetry/opentelemetry-collector/blob/main/receiver/hostmetricsreceiver/metadata.yaml) that lists all of the metrics it is capable of producing, and all of the tags that those metrics may receive. We should generate a config like the one in the above example, using this yaml file. 

In particular, for any metric produced by this collector, we should do this transform on any label that is not identified by an enum field. (As an example, [here is what the enum of filesystem.device looks like](https://github.com/open-telemetry/opentelemetry-collector/blob/main/receiver/hostmetricsreceiver/metadata.yaml#L26-L28).) *Exception: network.state (for the metric system.network.connections) is a well-bounded list of values, we should not aggregate those away.*

### Extract static, well-known attribute groups into metric names

Extract certain labels/attributes into their own metrics. Include the value of the attribute in the new metric’s name. (For example, every data point in a system.memory.usage metric with a key of state and a value of buffered would be moved to a new metric with the name system.memory.usage.buffered.)

This heuristic operates only on metrics with given, specific names (for example, `system.memory.usage`). All other metrics will pass through the processor unchanged.

For example, we would convert the above metrics stream into one that looks like this instead:

```
metrics:{
  name:"system.memory.usage.used"
  description:"Bytes of memory in use."
  unit:"By"
  int_sum:{
    data_points:{
      time_unix_nano:1623277287515079370
      value:217608192
    }
    aggregation_temporality:AGGREGATION_TEMPORALITY_CUMULATIVE
  }
}
metrics:{
  name:"system.memory.usage.free"
  description:"Bytes of memory in use."
  unit:"By"
  int_sum:{
    data_points:{
      time_unix_nano:1623277287515079370
      value:79982592
    }
    aggregation_temporality:AGGREGATION_TEMPORALITY_CUMULATIVE
  }
}
metrics:{
  name:"system.memory.usage.buffered"
  description:"Bytes of memory in use."
  unit:"By"
  int_sum:{
    data_points:{
      time_unix_nano:1623277287515079370
      value:28291072
    }
    aggregation_temporality:AGGREGATION_TEMPORALITY_CUMULATIVE
  }
}
metrics:{
  name:"system.memory.usage.cached"
  description:"Bytes of memory in use."
  unit:"By"
  int_sum:{
    data_points:{
      time_unix_nano:1623277287515079370
      value:703111168
    }
    aggregation_temporality:AGGREGATION_TEMPORALITY_CUMULATIVE
  }
}
metrics:{
  name:"system.memory.usage.slab_reclaimable"
  description:"Bytes of memory in use."
  unit:"By"
  int_sum:{
    data_points:{
      time_unix_nano:1623277287515079370
      value:41844736
    }
    aggregation_temporality:AGGREGATION_TEMPORALITY_CUMULATIVE
  }
}
metrics:{
  name:"system.memory.usage.slab_unreclaimable"
  description:"Bytes of memory in use."
  unit:"By"
  int_sum:{
    data_points:{
      time_unix_nano:1623277287515079370
      value:41148416
    }
    aggregation_temporality:AGGREGATION_TEMPORALITY_CUMULATIVE
  }
}
```

When they are ingested, these would result in a single event in Honeycomb.

```json
{
  "Timestamp": "2021-06-09T22:21:27.51507937Z",
  "host.name": "vm2",
  "service.name": "webserver",
  "system.memory.usage.used": 217608192,
  "system.memory.usage.free": 79982592,
  "system.memory.usage.buffered": 28291072,
  "system.memory.usage.cached": 703111168,
  "system.memory.usage.slab_reclaimable": 41844736,
  "system.memory.usage.slab_unreclaimable": 41148416,
}
```

This type of transform is be supported by the [*metricstransform* processor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/metricstransformprocessor#readme) in the contrib repo for OpenTelemetry collector. (Though, it’s quite verbose, and there is a bug with update mode (https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/4194) that requires an additional filtering step afterwards.) For example, it is possible to support the above example with a transform pipeline configured like this:

```yaml
processors:
  metricstransform:
    transforms:
      - include: system.memory.usage
        experimental_match_labels: {state: "used"}
        action: insert
        new_name: "system.memory.usage.used"
      - include: system.memory.usage
        experimental_match_labels: {state: "free"}
        action: insert
        new_name: "system.memory.usage.free"
     - include: system.memory.usage
        experimental_match_labels: {state: "buffered"}
        action: insert
        new_name: "system.memory.usage.buffered"
       - include: system.memory.usage
        experimental_match_labels: {state: "cached"}
        action: insert
        new_name: "system.memory.usage.cached"
      - include: system.memory.usage
        experimental_match_labels: {state: "slab_reclaimable"}
        action: insert
        new_name: "system.memory.usage.slab_reclaimable"
      - include: system.memory.usage
        experimental_match_labels: {state: "slab_unreclaimable"}
        action: insert
        new_name: "system.memory.usage.slab_unreclaimable"
  filter:
    metrics:
      exclude:
        match_type: strict
        metric_names:
          - system.memory.usage
          
service:
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: [metricstransform, filter]
      exporters: [otlp]
```

The hostmetrics reciever [comes with a metadata.yaml file](https://github.com/open-telemetry/opentelemetry-collector/blob/main/receiver/hostmetricsreceiver/metadata.yaml) that lists all of the metrics it is capable of producing, and all of the tags that those metrics may receive. We should generate a config like the one in the above example, using this yaml file. 

In particular, for any metric produced by this collector, we should do this transform on any label that is identified by an enum field. (As an example, [here is what the enum of mem.state looks like](https://github.com/open-telemetry/opentelemetry-collector/blob/main/receiver/hostmetricsreceiver/metadata.yaml#L21-L24).) If there is more than one label with an enum, we should apply them in a consistent order defined by their order in the yaml file.... for example, `system.paging.operations` with `{direction: page_in, type: minor}` should be repackaged into `system.paging.operations.page_in.minor` rather than `system.paging.operations.minor.page_in`.
