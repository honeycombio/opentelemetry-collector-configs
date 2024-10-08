# Reduce Processor

<!-- status autogenerated section -->
| Status        |           |
| ------------- |-----------|
| Stability     | [development]: logs   |
| Distributions | [] |
| Issues        | [![Open issues](https://img.shields.io/github/issues-search/open-telemetry/opentelemetry-collector-contrib?query=is%3Aissue%20is%3Aopen%20label%3Aprocessor%2Freduce%20&label=open&color=orange&logo=opentelemetry)](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues?q=is%3Aopen+is%3Aissue+label%3Aprocessor%2Freduce) [![Closed issues](https://img.shields.io/github/issues-search/open-telemetry/opentelemetry-collector-contrib?query=is%3Aissue%20is%3Aclosed%20label%3Aprocessor%2Freduce%20&label=closed&color=blue&logo=opentelemetry)](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues?q=is%3Aclosed+is%3Aissue+label%3Aprocessor%2Freduce) |
| [Code Owners](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/CONTRIBUTING.md#becoming-a-code-owner)    | [@MikeGoldsmith](https://www.github.com/MikeGoldsmith), [@codeboten](https://www.github.com/codeboten) |

[development]: https://github.com/open-telemetry/opentelemetry-collector#development
<!-- end autogenerated section -->

This processor is used to combine related log events together based on a set of shared attributes.

## Configuration Options

| Name | Description | Required | Default Value | 
| - | - | - | - |
| group_by | The list of attribute names used to group and aggregate log records. At least one attribute name is required. | Yes | `none` |
| reduce_timeout | The amount of time to wait after the last log record was received before an aggreated log record should be considered complete. | No | `10s` |
| max_reduce_timeout | The maximum amount of time an aggregated log record can be stored in the cache before it should be considered complete. | No | `60s` |
| max_reduce_count | The maximum number of log records that can be aggregated together. If the maximum is reached, the current aggregated log record is considered complete and a new aggregated log record is created. | No | `100` |
| cache_size | The maximum number of entries that can be stored in the cache. | No | `10000` |
| merge_strategies | A map of attribute names to a custom merge strategies. If an attribute is not found in the map, the default merge strategy of `First` is used. | No | `none` |
| reduce_count_attribute | The the attribute name used to store the count of log records on the aggregated log record'. If empty, the count is not stored. | No | `none` |
| first_seen_attribute | The attribute name used to store the timestamp of the first log record in the aggregated log record. If empty, the last seen time is not stored. | No | `none` |
| last_seen_attribute | The attribute name used to store the timestamp of the last log record in the aggregated log record. If empty, the last seen time is not stored. | No | `none` |

### Merge Strategies

| Name | Description |
| - | - |
| First | Keeps the first non-empty value. |
| Last | Keeps the last non-empty value. |
| Array | Combines multiple values into an array. |
| Concat | Concatenates each non-empty value together with a comma `,`. |

### Example configuration

The following is the minimal configuration of the processor:

```yaml
reduce:
  group_by:
    - "host.name"
```

The following is a complete configuration of the processor:

```yaml
reduce:
  group_by:
    - "host.name"
  reduce_timeout: 10s
  max_reduce_timeout: 60s
  max_reduce_count: 100
  cache_size: 10000
  merge_strategies:
    "some-attribute": first
    "another-attribute": last
  reduce_count_attribute: reduce_count
  first_seen_attribute: first_timestamp
  last_seen_attribute: last_timestamp
```
