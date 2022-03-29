#!/bin/bash

# Build clickhouse-operator docker images
TAG="${TAG:-latest}"
IMAGE_PREFIX="${IMAGE_PREFIX:-radondb}"

CUR_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
source "${CUR_DIR}/image_build_operator.sh"

CUR_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
source "${CUR_DIR}/image_build_metrics_exporter.sh"
