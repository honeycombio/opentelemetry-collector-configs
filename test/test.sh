#!/bin/bash

set -e

output_file=$(mktemp ./artifacts/test-output-XXXXXX)
orig_config_file="./artifacts/honeycomb-metrics-config.yaml"
test_config_file=$(mktemp /tmp/test-config-XXXXXX)

# make some modifications to config to make it testable
yq -y '. * {
  exporters: {
    file: {path: "'$output_file'"}
  },
  receivers: { 
    hostmetrics: {
      collection_interval: "1s"
    }
  },
  service: {
    pipelines: {
      metrics: { exporters: ["file"] }
    }
  }
}' < $orig_config_file > $test_config_file

touch $output_file
echo "writing JSON OTLP to $output_file."

# run otelcol in the background until we have some data to look at
echo -n "running otelcol until we have data..."

output_line_count () {
  wc -l $output_file | awk '{print $1}'
}
./build/otelcol_hny_$(go env GOOS)_$(go env GOARCH) --config $test_config_file >/dev/null 2>&1 &
otelcol_pid=$!

while [ $(output_line_count) -lt 1 ]
do
sleep 1
done

# wait for otelcol to cleanly exit
kill $otelcol_pid
wait $otelcol_pid
echo "success!"

# assert that metric names are correct
found_names=$(jq -r '
  .resourceMetrics[] |
  .scopeMetrics[] |
  .metrics[] |
  .name' < $output_file)
echo "checking that we only generated metrics with allowed names..."
allowed_names=$(<./test/allowed_metric_names.txt)

for metric_name in $found_names
do
  if [[ ${allowed_names} != *"$metric_name"* ]]; then
    echo "found disallowed metric $metric_name"
    exit 1
  fi
done

echo "ensuring we have at least ~40 distinct metrics (less than that is an indication something is wrong)..."
metric_count=$(echo $found_names | wc -w)
if (( ${metric_count} < 40 )); then
  echo "only found ${metric_count} distinct metrics"
  exit 1
fi

unique_timestamps=$(jq -sr '
    .[0].resourceMetrics[] |
    .scopeMetrics[] |
    .metrics[] |
    .sum // .gauge // .summary // .histogram |
    .dataPoints[] |
    .timeUnixNano' < $output_file |
  sort |
  uniq)

# assert that all timestamps are truncated to the nearest 1 second
for timestamp in $unique_timestamps; do
  if [[ $timestamp != *000000000 ]]; then
    echo "found a timestamp that is not truncated to the nearest 1 second: $timestamp"
    exit 1
  fi
done

# assert that there are, at maximum, two different timestamps across all the datapoints for a single batch
echo "checking that all metric datapoints have at most two distinct timestamps..."
unique_timestamp_count=$(echo $unique_timestamps | wc -l)
if (( $unique_timestamp_count > 2 )); then
  echo "found $unique_timestamp_count timestamps, but we needed 2 or less"
  exit 1
fi

echo "success!"
