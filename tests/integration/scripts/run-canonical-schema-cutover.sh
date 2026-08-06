#!/usr/bin/env bash

set -euo pipefail

# Reproduce M16 against a fresh ClickHouse 25.5.6 instance. The Collector image
# is built from the sibling checkout so every test uses the same writer/migrator
# revision as the direct baseline verification.
siginsight_root="$(git rev-parse --show-toplevel)"
collector_root="${COLLECTOR_ROOT:-$(cd "${siginsight_root}/../OtelCollector" && pwd)}"
collector_image="${SIGINSIGHT_OTEL_COLLECTOR_IMAGE:-siginsight-otel-collector:m16-test}"

if [[ ! -f "${collector_root}/cmd/siginsightotelcollector/main.go" ]]; then
	printf 'Collector repository not found at %s. Set COLLECTOR_ROOT.\n' "${collector_root}" >&2
	exit 1
fi

if [[ -z "${SIGINSIGHT_OTEL_COLLECTOR_IMAGE:-}" ]]; then
	make -C "${collector_root}" build
	docker build \
		--build-arg TARGETOS=linux \
		--build-arg TARGETARCH=amd64 \
		--tag "${collector_image}" \
		-f "${collector_root}/cmd/siginsightotelcollector/Dockerfile" \
		"${collector_root}"
fi

export SIGINSIGHT_OTEL_COLLECTOR_IMAGE="${collector_image}"
export SIGINSIGHT_OTEL_COLLECTOR_ROOT="${collector_root}"

make -C "${collector_root}" test-migration-integration

cd "${siginsight_root}/tests/integration"
uv run pytest --clickhouse-version=25.5.6 -vv -s \
	src/compat/03_litequery_collector_collaboration.py \
	src/compat/04_materialized_catalog.py \
	src/ttl/01_ttl.py
