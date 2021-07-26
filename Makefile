create_config: ./config-generator.jq ./hostmetrics-receiver-metadata.yaml
	yq -y -f ./config-generator.jq < ./hostmetrics-receiver-metadata.yaml > ./compact-config.yaml