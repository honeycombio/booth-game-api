FROM golang:1.21 as build
ARG  OTEL_VERSION=0.90.1
WORKDIR /app
RUN go install go.opentelemetry.io/collector/cmd/builder@v${OTEL_VERSION}
COPY builder-config.yaml .
RUN CGO_ENABLED=0 builder --config=builder-config.yaml --output-path=/app

FROM cgr.dev/chainguard/static:latest
COPY --from=build /app/otelcol-custom /
COPY config.yaml /
EXPOSE 4317/tcp 4318/tcp 13133/tcp

CMD ["/otelcol-custom", "--config=/config.yaml"]