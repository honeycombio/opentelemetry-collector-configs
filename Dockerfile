FROM golang:1.24 AS otelcontribcol
WORKDIR /go/src
ENV CGO_ENABLED=0

ADD ./builder-config.yaml ./
RUN go install go.opentelemetry.io/collector/cmd/builder@v0.122.1
RUN builder --config builder-config.yaml

# use the official upstream image for the opampsupervisor
FROM ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector-opampsupervisor AS opampsupervisor

FROM alpine:latest

ARG USER_UID=10001
ARG USER_GID=10001
USER ${USER_UID}:${USER_GID}

COPY --from=opampsupervisor --chmod=755 /usr/local/bin/opampsupervisor /opampsupervisor
COPY --from=otelcontribcol --chmod=755 /go/src/supervised-collector /otelcol-contrib
WORKDIR /var/lib/otelcol/supervisor
ENTRYPOINT ["/opampsupervisor"]
