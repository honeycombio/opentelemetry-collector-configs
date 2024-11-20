FROM golang:1.23 AS build

WORKDIR /go/src
ADD ./build ./

RUN CGO_ENABLED=0 go build -o otelcol-hny ./...  

FROM alpine:3.19 AS certs
RUN apk --update add ca-certificates

FROM scratch

ARG USER_UID=10001
USER ${USER_UID}

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build --chmod=755 /go/src/otelcol-hny /otelcol-hny
ENTRYPOINT ["/otelcol-hny"]
CMD ["--config", "/etc/otelcol-contrib/config.yaml"]
EXPOSE 4317 55678 55679
