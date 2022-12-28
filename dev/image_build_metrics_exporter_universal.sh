#!/bin/bash

# Universal docker image builder.
# Should be called from image_build_operator.sh or image_build_metrics_exporter.sh

set -e

# Declared from the called script:
# TAG
# DOCKER_IMAGE
# DOCKERFILE_DIR

CUR_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
SRC_ROOT="$(realpath "${CUR_DIR}/..")"
MINIKUBE="${MINIKUBE:-no}"
DOCKERHUB_LOGIN="${DOCKERHUB_LOGIN}"
DOCKERFILE_DIR="${SRC_ROOT}/dockerfile/metrics-exporter"
DOCKERFILE="${DOCKERFILE_DIR}/Dockerfile"
DOCKERHUB_PUBLISH="${DOCKERHUB_PUBLISH:-no}"

# Source-dependent options
source "${CUR_DIR}/go_build_config.sh"

# Build image with Docker
if [[ "${MINIKUBE}" == "yes" ]]; then
    # We'd like to build for minikube
    eval "$(minikube docker-env)"
fi

if ! docker run --rm --privileged multiarch/qemu-user-static --reset -p yes; then
    sudo apt-get install -y qemu binfmt-support qemu-user-static
    docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
fi

if [[ "0" == $(docker buildx ls | grep -E 'linux/arm.+\*' | grep -E 'running|inactive') ]]; then
    docker buildx create --use --name multi-platform --platform=linux/amd64,linux/arm64
fi

DOCKER_CMD="docker buildx build --progress plain"
if [[ "${DOCKER_IMAGE}" =~ ":dev" || "yes" == "${MINIKUBE}" ]]; then
    DOCKER_CMD="${DOCKER_CMD} --output type=image,name=${DOCKER_IMAGE} --platform=linux/amd64"
else
    DOCKER_CMD="${DOCKER_CMD} --platform=linux/amd64,linux/arm64"
fi

DOCKER_CMD="${DOCKER_CMD} --build-arg VERSION=${VERSION:-dev} --build-arg RELEASE=${RELEASE:-1}"

if [[ "" != "${GCFLAGS}" ]]; then
    DOCKER_CMD="${DOCKER_CMD} --build-arg GCFLAGS='${GCFLAGS}'"
fi

if [[ "${DOCKERHUB_PUBLISH}" == "yes" ]]; then
    DOCKER_CMD="${DOCKER_CMD} --push"
fi

#DOCKER_CMD="${DOCKER_CMD} -t ${DOCKER_IMAGE} -f ${DOCKERFILE} ${SRC_ROOT}"
DOCKER_CMD="${DOCKER_CMD} -t ${TAG} -f ${DOCKERFILE} ${SRC_ROOT}"

if [[ "${DOCKERHUB_PUBLISH}" == "yes" ]]; then
    if [[ -n "${DOCKERHUB_LOGIN}" ]]; then
        echo "Dockerhub login specified: '${DOCKERHUB_LOGIN}', perform login"
        docker login -u "${DOCKERHUB_LOGIN}"
    fi
fi

if ${DOCKER_CMD}; then
    echo "ALL DONE. Docker image published."
else
    echo "FAILED docker build! Abort."
    exit 1
fi