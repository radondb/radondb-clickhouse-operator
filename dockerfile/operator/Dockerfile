# ===================
# ===== Builder =====
# ===================

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

# Build operator binary with explicitly specified output
RUN OPERATOR_BIN=/tmp/clickhouse-operator bash -xe ./dev/go_build_operator.sh

# ======================
# ===== Image Base =====
# ======================

FROM --platform=${BUILDPLATFORM} registry.access.redhat.com/ubi8/ubi-minimal AS image-base
RUN microdnf update && microdnf clean all

MAINTAINER "RadonDB <support@radondb.com>"

LABEL name="ClickHouse operator" \
      maintainer="support@radondb.com" \
      vendor="RadonDB" \
      version="${VERSION:-dev}" \
      release="${RELEASE:-1}" \
      summary="ClickHouse operator" \
      description="ClickHouse operator operates ClickHouse clusters in kubernetes"

ADD LICENSE /licenses/

# set timezone
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
ENV TZ=Asia/Shanghai

WORKDIR /

# Add config files from local source dir into image
ADD config/config.yaml   /etc/clickhouse-operator/
ADD config/conf.d/*      /etc/clickhouse-operator/conf.d/
ADD config/config.d/*    /etc/clickhouse-operator/config.d/
ADD config/templates.d/* /etc/clickhouse-operator/templates.d/
ADD config/users.d/*     /etc/clickhouse-operator/users.d/

# Copy clickhouse-operator binary into operator image from builder
COPY --from=builder /tmp/clickhouse-operator .

USER nobody

# Run /clickhouse-operator -alsologtostderr=true -v=1
# We can specify additional options, such as:
#   --config=/path/to/config
#   --kube-config=/path/to/kubeconf
ENTRYPOINT ["/clickhouse-operator"]
CMD ["-logtostderr=true", "-v=1"]
#CMD ["-alsologtostderr=true", "-v=1"]
