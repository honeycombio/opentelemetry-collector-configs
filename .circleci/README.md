# Timestamp Processor Build

The timestampprocessor relies on the `honeycombio/cci-go-yq` docker image.
This image is built from `images/Dockerfile`.

Whenever bumping the collector libraries the version for `OTC_BUILDER_VERSION` within the Dockerfile must be updated to match the collector library version.
Once the Dockerfile is updated, the image must be built locally and pushed to DockerHub with a new patch version. 
Once the image is pushed, `config.yaml` should be updated to use the new tag.