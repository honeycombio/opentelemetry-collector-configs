#!/bin/bash

yq -Y -f hostmetrics-transform.jq < hostmetrics-receiver-metadata.yaml > config.yaml