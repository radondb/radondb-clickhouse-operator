#!/bin/bash

set -e

CUR_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
SRC_ROOT="$(realpath "${CUR_DIR}/..")"

# Externally configurable build-dependent options
TAG="${TAG:-latest}"
IMAGE_PREFIX="${IMAGE_PREFIX:-radondb}"

DOCKER_IMAGE="${IMAGE_PREFIX}/chronus-metrics-operator:${TAG}"
DOCKERFILE_DIR="${SRC_ROOT}/dockerfile/metrics-exporter"

source "${CUR_DIR}/image_build_universal.sh"
