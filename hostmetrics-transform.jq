.labels as $labels |

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
  # only transform metrics that have non-enum labels
  select(.non_enum_labels | length > 0)
  | {
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

# rename labels transforms:
[ 
  $metrics[] |
  . as $metric |
  # select(.enum_labels | length > 1) | # temporary, remove me
  .enum_labels[] |
  .value as $label_name |
  .enum[] |
  {
    "include": $metric.name,
    "experimental_match_labels": {
      ($label_name): .
    },
    "action": "insert",
    "new_name": "\($metric.name).\(.)"
  }
] as $rename_labels_transforms |

# output formatted for otel collector configuration
{
	"processors": {
    "metricstransform": {
      "transforms": (
        $aggregate_labels_transforms +
        $rename_labels_transforms
      )
    }
  }
}