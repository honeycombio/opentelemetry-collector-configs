#!/bin/bash

yq -y -f hostmetrics-transform.jq < hostmetrics-receiver-metadata.yaml > config.yaml