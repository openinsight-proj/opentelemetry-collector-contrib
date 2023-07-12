FROM alpine:latest as prep
RUN apk --update add ca-certificates

RUN mkdir -p /tmp

FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.19 AS builder
ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH
COPY ./cmd/otelcontribcol_${TARGETOS}_${TARGETARCH} /otelcol-contrib

FROM scratch

ARG USER_UID=10001
USER ${USER_UID}

COPY --from=prep /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /otelcol-contrib /otelcol-contrib
EXPOSE 4317 55680 55679
ENTRYPOINT ["/otelcol-contrib"]
CMD ["--config", "/etc/otel/config.yaml"]