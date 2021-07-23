#!/bin/bash

yq -y -f ./config-generator/build.jq < ./config-generator/hostmetrics-receiver-metadata.yaml > ./compact-config.yaml