ARG ARCH="amd64"
ARG OS="linux"
FROM quay.io/prometheus/busybox-${OS}-${ARCH}:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

ARG ARCH="amd64"
ARG OS="linux"
COPY .build/${OS}-${ARCH}/avalanche /bin/avalanche
COPY .build/${OS}-${ARCH}/mtypes /bin/mtypes

EXPOSE      9101
USER        nobody
ENTRYPOINT  [ "/bin/avalanche" ]
