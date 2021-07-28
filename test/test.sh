#!/bin/bash

set -e

output_file="./test/tmp-test-output.json"
orig_config_file="./compact-config.yaml"
test_config_file="./test/tmp-test-config.yaml"

# make some modifications to config to make it testable
rm -f $test_config_file
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

rm -f $output_file
touch $output_file

# run otelcol in the background until we have some data to look at
echo -n "running otelcol until we have data..."

output_line_count () {
  wc -l $output_file | awk '{print $1}'
}
./build/otelcol-hny --config $test_config_file >/dev/null 2>&1 &
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
echo "checking that we only generated metrics with allowed names..."
allowed_names=$(<./test/allowed_metric_names.txt)
for metric_name in $(jq -r '
  .resourceMetrics[] |
  .instrumentationLibraryMetrics[] |
  .metrics[] |
  .name')
do
  if [[ ${allowed_names} != *"$metric_name"* ]]; then
    echo "found disallowed metric $metric_name"
    exit 1
  fi
done < $output_file

echo "success!"
