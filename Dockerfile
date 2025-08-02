FROM golang:1.24.5-bookworm@sha256:ef8c5c733079ac219c77edab604c425d748c740d8699530ea6aced9de79aea40
RUN apt-get update && \
    apt-get install -y ca-certificates && \
    update-ca-certificates

WORKDIR /opt/prometheus-llm-proxy

COPY . .
RUN go mod tidy
RUN go build .

FROM debian:bookworm-20250721-slim@sha256:2424c1850714a4d94666ec928e24d86de958646737b1d113f5b2207be44d37d8
COPY --from=0 /opt/prometheus-llm-proxy/prometheus-llm-proxy /opt/prometheus-llm-proxy

ENTRYPOINT ["/opt/prometheus-llm-proxy"]