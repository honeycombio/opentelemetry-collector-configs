# OPENTELEMETRY-COLLECTOR CONFIG GENERATOR
# Written as a jq program -- refer to https://stedolan.github.io/jq/
# However, since input and output files are probably yaml, I would
# recommend using yq instead: https://kislyuk.github.io/yq/#

# Usage: yq -y -f config-generator.jq < input.yaml > output.yaml
# Input: a file that describes all of the metrics & labels that are
# created by the hostmetrics receiver.
# Output: a configuration file for opentelemetry-collector. See README
# for details.

# Save label list, as a key-value object, to $labels:
(
  .labels *
  # merge a hardcoded list of TCP states into the network.state label
  # so that they don't get pre-aggregated away. (These states are
  # enumerated in https://datatracker.ietf.org/doc/html/rfc793.)
  {
    "network.state": (
      {
        "enum": [
          "CLOSE_WAIT",
          "CLOSED",
          "CLOSING",
          "DELETE",
          "ESTABLISHED",
          "FIN_WAIT_1",
          "FIN_WAIT_2",
          "LAST_ACK",
          "LISTEN",
          "SYN_RECEIVED",
          "SYN_SENT",
          "TIME_WAIT"
        ]
      }
    )
  }
) as $labels |

# Save metrics list, as an array, to $metrics:
.metrics |
[
  to_entries[] |
  # for each metric, extract label details from $labels.
  # separate labels into enum_labels (which we want to rename)
  # and non_enum_labels (which we want to aggregate away)
  {
    "name": .key,
    "enum_labels": [
      .value.labels[]? |
      $labels[.] |
      select(has("enum"))
    ],
    "non_enum_labels": [
      .value.labels[]? |
      $labels[.] |
      select(has("enum") | not)
    ]
  }
] as $metrics |

# put together the config section that's responsible for
# aggregating non_enum labels, and save that to
# $aggregate_labels_transforms:
[
  $metrics[] |
  # transform the metrics that only have non-enum labels
  select(.non_enum_labels | length > 0) |
  select(.enum_labels | length == 0) |
  {
    "include": .name,
    "action": "update",
    "operations": [
      {
        "action": "aggregate_labels",
        "aggregation_type": "sum",
        # remove everything except for the enum labels
        "label_set": [ .enum_labels[] | .value ]
      }
    ]
  }
] as $aggregate_labels_transforms |

# here is a function that, given a single metric, will determine the renames
# that need to occur so that all the enum labels get extracted:
def build_rename_rule_from_metric: (
  .name as $old_name |
  .enum_labels[0] as $current |
  .enum_labels[1:] as $rest |
  [
    $current.enum[] |
    "\($old_name).\(.)" as $new_name |
    [
      {
        "old_name": $old_name,
        "new_name": $new_name,
        "label_key": $current.value,
        "label_value": .,
        "aggregate_to": [$rest[] | .value]
      }
    ] +
    # if there is more than one enum label on this metric, then recursively
    # create rules to extract new metrics from _those_ timeseries from the
    # metric we just created.
    if (($rest | length) > 0) 
    then (
      { "name": $new_name, "enum_labels": $rest } |
      build_rename_rule_from_metric
    )
    # base case:
    else []
    end |
    flatten(1)
  ]
);

# build a list of metric renames
# unique timeseries out of the enum labels for each metric, and save
# that to $metric_rename_rules:
[
  $metrics[] |
  # only transform metrics that have enum labels
  select(.enum_labels | length > 0) |
  build_rename_rule_from_metric
] |
flatten(2) |
. as $metric_rename_rules |

# here are the metricstransform transforms for renaming:
[
  $metric_rename_rules[] |
  {
    "include": .old_name,
    "experimental_match_labels": { (.label_key): .label_value },
    "action": "insert",
    "new_name": .new_name,
    "operations": [
      {
        "action": "aggregate_labels",
        "aggregation_type": "sum",
        "label_set": .aggregate_to
      }
    ]
  }
] as $metric_rename_transforms |

# we also need to build a filter rule to remove the metrics whose names
# we changed:
[ $metric_rename_rules[] | .old_name ] | unique as $metrics_to_exclude |

# build & output the final configuration:
{
  "exporters": {
    "otlp": {
      # $ENV pulls from the environment when the generator is running; // provides a "default" value:
      "endpoint": ($ENV.OTLP_ENDPOINT // "api.honeycomb.io:443"),
      "headers": {
        "x-honeycomb-team": "$HNY_API_KEY",
        "x-honeycomb-dataset": "$HNY_DATASET",
      }
    },
    "logging": {
      "loglevel": ($ENV.LOG_LEVEL // "info")
    }
  },
  "receivers": {
    "hostmetrics": {
      "collection_interval": ($ENV.COLLECTION_INTERVAL // "10s"),
      "scrapers": {
        "memory": {},
        "cpu": {},
        "load": {},
        "disk": {},
        "filesystem": {},
        "network": {},
        "paging": {},
        "processes": {}
      }
    }
  },
  "processors": {
    "resourcedetection": {
      "detectors": ["env", "system"]
    },
    "transform": {
      "error_mode": "ignore",
      "metric_statements": [
        {
          "context": "datapoint",
          "statements": [
            "set(time, TruncateTime(time, Duration(\"1s\")))"
          ]
        }
      ]
    },
    "batch": {
      "send_batch_size": 8192,
      "timeout": "200ms"
    },
    "metricstransform": {
      "transforms": (
        $aggregate_labels_transforms +
        $metric_rename_transforms
      )
    },
    "filter": {
      "metrics": {
        "exclude": {
          "match_type": "strict",
          "metric_names": $metrics_to_exclude
        }
      }
    }
  },
  "service": {
    "pipelines": {
      "metrics": {
        "receivers": ["hostmetrics"],
        "processors": ["metricstransform", "filter", "transform", "resourcedetection", "batch"],
        "exporters": ["logging", "otlp"]
      }
    }
  }
}