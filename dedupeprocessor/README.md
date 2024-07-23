# Dedupe Processor

*Status*: Experimental

The dedupe processor deduplicates log records by dropping records that match recently seen log records.
A hash is created for a log record using it's attributes (resource, scope and record) along with it's log level (severity) and body.
The hash is stored in a LRU with a configurable expiry.
Log records that produce the same hash are dropped.

### Configuration Options

| Name        | Description                                                                            | Default Value |
| ----------- | -------------------------------------------------------------------------------------- | ------------- |
| ttl         | The TTL for log record hashes to live in the cache for expressed as a `time.Duration`. | 30 seconds    |
| max_entries | The maximum number of entries for the LRU cache.                                       | 1000          |

### Example configuration

The following is an example of configuring the processor:

```yaml
dedupe:
  ttl: 1m # 1 minute
  max_entries: 10_000
```
