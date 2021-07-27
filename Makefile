all: config collector
config: compact-config.yaml
collector: build/otelcol-hny

test: test/test.sh collector 
	./test/test.sh

# generate a configuration file for otel-collector that results in a favorable repackaging ratio
compact-config.yaml: config-generator.jq vendor/hostmetrics-receiver-metadata.yaml
	yq -y -f ./config-generator.jq < ./vendor/hostmetrics-receiver-metadata.yaml > ./compact-config.yaml

# copy hostmetrics metadata yaml file from the OpenTelemetry Collector repository, and prepend a note saying it's vendored
vendor/hostmetrics-receiver-metadata.yaml:
	REMOTE_PATH='https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector/main/receiver/hostmetricsreceiver/metadata.yaml'; \
	curl $$REMOTE_PATH | sed "1s|^|# DO NOT EDIT! This file is vendored from $${REMOTE_PATH}"$$'\\\n\\\n|' > vendor/hostmetrics-receiver-metadata.yaml

build/otelcol-hny: builder-config.yaml
	opentelemetry-collector-builder --output-path=build --name=hny-otel --config=builder-config.yaml

clean:
	rm vendor/* build/* compact-config.yaml test/tmp-*

.PHONY: all config collector clean test
