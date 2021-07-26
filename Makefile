config: config-generator.jq vendor/hostmetrics-receiver-metadata.yaml
	yq -y -f ./config-generator.jq < ./vendor/hostmetrics-receiver-metadata.yaml > ./compact-config.yaml

# copy hostmetrics metadata yaml file from the OpenTelemetry Collector repository, and prepend a note saying it's vendored
vendor/hostmetrics-receiver-metadata.yaml:
	REMOTE_PATH='https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector/main/receiver/hostmetricsreceiver/metadata.yaml'; \
	curl $$REMOTE_PATH | sed "1s|^|# DO NOT EDIT! This file is vendored from $${REMOTE_PATH}"$$'\\\n\\\n|' > vendor/hostmetrics-receiver-metadata.yaml

clean:
	rm vendor/* compact-config.yaml

.PHONY: config clean