#!/bin/sh
#
# This script will attempt to mirror the host paths by using volumes for the
# following paths:
#   * ${PWD}
#
# You can add additional volumes (or any docker run options) using
# the ${JQ_OPTIONS} environment variable.
#
# You can set a specific image and tag, such as "lscr.io/linuxserver/yq:2.11.1-ls1"
# using the $YQ_IMAGE_TAG environment variable (defaults to "lscr.io/linuxserver/yq:latest")
#

set -e

# Setup volume mounts for compose config and context
if [ "${PWD}" != '/' ]; then
    VOLUMES="-v ${PWD}:${PWD}"
fi

# Only allocate tty if we detect one
if [ -t 0 ] && [ -t 1 ]; then
    DOCKER_RUN_OPTIONS="${DOCKER_RUN_OPTIONS} -t"
fi

# Always set -i to support piped and terminal input in run/exec
DOCKER_RUN_OPTIONS="${DOCKER_RUN_OPTIONS} -i"

# shellcheck disable=SC2086
exec docker run --rm ${DOCKER_RUN_OPTIONS} ${JQ_OPTIONS} ${VOLUMES} -w "${PWD}" --entrypoint jq "${YQ_IMAGE_TAG:-lscr.io/linuxserver/yq:latest}" "$@"
