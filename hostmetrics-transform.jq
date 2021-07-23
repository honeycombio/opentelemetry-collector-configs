(
  .labels + {
    "network.state": (
      .labels["network.state"] + {
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

.metrics |
[
	to_entries[] |
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

# aggregate labels transforms:
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

def rename: (
  .name as $old_name |
  .enum_labels[0] as $current |
  .enum_labels[1:] as $rest |
  [
    $current.enum[] |
    "\($old_name).\(.)" as $new_name |
    [
      {
        "include": $old_name,
        "experimental_match_labels": { ($current.value): . },
        "action": "insert",
        "new_name": $new_name,
        "operations": [
          {
            "action": "aggregate_labels",
            "aggregation_type": "sum",
            "label_set": [$rest[] | .value]
          }
        ]
      }
    ] +
    if (($rest | length) > 0)
    then (
      # recurse
      {
        "name": $new_name,
        "enum_labels": $rest
      } |
      rename
    )
    else [] # base case
    end |
    flatten(1)
  ]
);

# rename labels transforms:
[
  $metrics[] |
  # only transform metrics that have enum labels
  select(.enum_labels | length > 0) |
  rename
] |
flatten(2) |
. as $rename_labels_transforms |

# output formatted for otel collector configuration
{
  "exporters": {
    "otlp": {
      "endpoint": "api.honeycomb.io:443",
      "headers": {
        "x-honeycomb-team": "$HNY_API_KEY",
        "x-honeycomb-dataset": "$HNY_DATASET",
      }
    },
    "logging": {
      # "logLevel": "debug"
    }
  },
  "receivers": {
    "hostmetrics": {
      "collection_interval": "2s",
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
    "metricstransform": {
      "transforms": (
        $aggregate_labels_transforms +
        $rename_labels_transforms
      )
    },
    "filter": {
      "metrics": {
        "exclude": {
          "match_type": "strict",
          "metric_names": (
            [
              $metrics[] |
              select(.enum_labels | length > 0) |
              .name
            ] +
            [
              $metrics[] |
              select(.enum_labels | length > 1) |
              "\(.name).\(.enum_labels[0].enum[])" # TODO: this only works at depth 2, which is fine for now
            ]
          )
        }
      }
    },
    "batch": {}
  },
  "service": {
    "pipelines": {
      "metrics": {
        "receivers": ["hostmetrics"],
        "processors": ["metricstransform", "filter", "batch"],
        "exporters": ["logging", "otlp"]
      }
    }
  }
}