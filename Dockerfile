FROM alpine:latest as prep
RUN apk --update add ca-certificates

COPY ./ ./

RUN make -j2 gomoddownload && make install-tools && make otelcontribcol

RUN mkdir -p /tmp

FROM scratch

ARG USER_UID=10001
USER ${USER_UID}

COPY --from=prep /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=prep ./cmd/otelcontribcol /otelcol-contrib
EXPOSE 4317 55680 55679
ENTRYPOINT ["/otelcol-contrib"]
CMD ["--config", "/etc/otel/config.yaml"]