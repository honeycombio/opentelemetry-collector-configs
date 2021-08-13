all: config collector
config: artifacts/honeycomb-metrics-config.yaml
collector: build/otelcol-hny

test: test/test.sh build/otelcol-hny artifacts/honeycomb-metrics-config.yaml
	./test/test.sh

# generate a configuration file for otel-collector that results in a favorable repackaging ratio
artifacts/honeycomb-metrics-config.yaml: config-generator.jq vendor-fixtures/hostmetrics-receiver-metadata.yaml
	mkdir -p ./artifacts
	yq -y -f ./config-generator.jq < ./vendor-fixtures/hostmetrics-receiver-metadata.yaml > ./artifacts/honeycomb-metrics-config.yaml

# copy hostmetrics metadata yaml file from the OpenTelemetry Collector repository, and prepend a note saying it's vendored
vendor-fixtures/hostmetrics-receiver-metadata.yaml:
	REMOTE_PATH='https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector/main/receiver/hostmetricsreceiver/metadata.yaml'; \
	curl $$REMOTE_PATH | sed "1s|^|# DO NOT EDIT! This file is vendored from $${REMOTE_PATH}"$$'\\\n\\\n|' > vendor-fixtures/hostmetrics-receiver-metadata.yaml

build/otelcol-hny: builder-config.yaml
	ls -al builder-config.yaml 
	opentelemetry-collector-builder --output-path=build --name=hny-otel --config=builder-config.yaml

clean:
	rm vendor-fixtures/* build/* compact-config.yaml test/tmp-*

.PHONY: all config collector clean test
