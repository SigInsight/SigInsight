ARG CLICKHOUSE_VERSION=25.5.6

FROM --platform=$BUILDPLATFORM golang:1.25.12 AS builder
WORKDIR /src

ARG TARGETOS
ARG TARGETARCH

COPY go.mod ./
COPY scripts/clickhouse/histogramquantile/main.go scripts/clickhouse/histogramquantile/main.go

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags='-s -w' -o /out/histogramQuantile ./scripts/clickhouse/histogramquantile

FROM clickhouse/clickhouse-server:${CLICKHOUSE_VERSION}

COPY --from=builder /out/histogramQuantile /opt/siginsight/bin/histogramQuantile

RUN chmod 755 /opt/siginsight/bin/histogramQuantile
