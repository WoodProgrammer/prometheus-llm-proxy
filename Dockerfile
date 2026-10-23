FROM golang:1.27.3-bookworm@sha256:972b4110ba0f47539a975c31e52c64765e1ff19f2d74469ae4a595c6676e80a5
RUN apt-get update && \
    apt-get install -y ca-certificates && \
    update-ca-certificates

WORKDIR /opt/prometheus-llm-proxy

COPY . .
RUN go mod download
RUN go build .

FROM debian:bookworm-slim@sha256:7ff6ba31527b9e336162fa061746c366c4b2032f4d2cff2078bea43f78efe3f8
RUN apt-get update && \
    apt-get install -y ca-certificates && \
    update-ca-certificates
COPY --from=0 /opt/prometheus-llm-proxy/prometheus-llm-proxy /opt/prometheus-llm-proxy

ENTRYPOINT ["/opt/prometheus-llm-proxy"]