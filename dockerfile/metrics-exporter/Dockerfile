# ===================
# ===== Builder =====
# ====================

FROM --platform=${BUILDPLATFORM} golang:1.17 AS builder

ARG VERSION
ARG RELEASE
ARG GCFLAGS
ARG TARGETOS
ARG TARGETARCH

# Install required packages
RUN sed -i s@/deb.debian.org/@/mirrors.aliyun.com/@g /etc/apt/sources.list \
    && apt-get update && apt-get install -y apt-utils gettext-base

# Reconstruct source tree inside docker
WORKDIR /clickhouse-operator
ADD . .

ENV GCFLAGS="${GCFLAGS}"
ENV GOOS="${TARGETOS}"
ENV GOARCH="${TARGETARCH}"

# ./vendor is excluded in .dockerignore, reconstruct it with 'mod' tool
ENV GOPROXY=https://goproxy.cn,direct
RUN GO111MODULE=on go mod tidy
RUN GO111MODULE=on go mod vendor

# Build operator binary with explicitly specified output
RUN METRICS_EXPORTER_BIN=/tmp/metrics-exporter ./dev/go_build_metrics_exporter.sh

# ======================
# ===== Image Base =====
# ======================

FROM --platform=${BUILDPLATFORM} registry.access.redhat.com/ubi8/ubi-minimal AS image-base
RUN microdnf update && microdnf clean all

MAINTAINER "RadonDB <support@radondb.com>"

LABEL name="ClickHouse operator. Metrics exporter" \
      maintainer="support@radondb.com" \
      vendor="RadonDB" \
      version="${VERSION:-dev}" \
      release="${RELEASE:-1}" \
      summary="Metrics Exporter" \
      description="Metrics exporter for RadonDB ClickHouse operator"

ADD LICENSE /licenses/

WORKDIR /

# Add config files from local source dir into image
ADD config/config.yaml   /etc/clickhouse-operator/
ADD config/conf.d/*      /etc/clickhouse-operator/conf.d/
ADD config/config.d/*    /etc/clickhouse-operator/config.d/
ADD config/templates.d/* /etc/clickhouse-operator/templates.d/
ADD config/users.d/*     /etc/clickhouse-operator/users.d/

# Copy clickhouse-operator binary into operator image from builder
COPY --from=builder /tmp/metrics-exporter .

USER nobody

# Run /metrics-exporter -alsologtostderr=true -v=1
# We can specify additional options, such as:
#   --config=/path/to/config
#   --kube-config=/path/to/kubeconf
ENTRYPOINT ["/metrics-exporter"]
CMD ["-logtostderr=true", "-v=1"]
#CMD ["-alsologtostderr=true", "-v=1"]
