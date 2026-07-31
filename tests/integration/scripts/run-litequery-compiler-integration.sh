#!/usr/bin/env bash

set -euo pipefail

siginsight_root="$(git rev-parse --show-toplevel)"
collector_root="${COLLECTOR_ROOT:-$(cd "${siginsight_root}/../OtelCollector" && pwd)}"
run_id="$$-${RANDOM}"
container="litequery-compiler-${run_id}"
work_dir="$(mktemp -d)"
collector_binary="${work_dir}/siginsight-otel-collector"

cleanup() {
	docker rm -f -v "${container}" >/dev/null 2>&1 || true
	rm -rf "${work_dir}"
}
trap cleanup EXIT

if [ ! -f "${collector_root}/go.mod" ]; then
	printf 'Collector repository not found at %s. Set COLLECTOR_ROOT.\n' "${collector_root}" >&2
	exit 1
fi

docker run -d --name "${container}" \
	-p 127.0.0.1::9000 \
	-e CLICKHOUSE_SKIP_USER_SETUP=1 \
	clickhouse/clickhouse-server:25.5.6 >/dev/null

clickhouse_port="$(docker port "${container}" 9000/tcp | awk -F: 'NR == 1 { print $NF }')"
if [ -z "${clickhouse_port}" ]; then
	printf 'Could not resolve temporary ClickHouse port.\n' >&2
	exit 1
fi

for _ in $(seq 1 60); do
	if docker exec "${container}" clickhouse-client --query 'SELECT 1' >/dev/null 2>&1; then
		break
	fi
	sleep 1
done

if ! docker exec "${container}" clickhouse-client --query 'SELECT 1' >/dev/null 2>&1; then
	printf 'Temporary ClickHouse did not become ready.\n' >&2
	exit 1
fi

go -C "${collector_root}" build -tags=remove_all_sd -o "${collector_binary}" ./cmd/siginsightotelcollector
for command in 'bootstrap' 'sync up' 'async up'; do
	"${collector_binary}" migrate ${command} --clickhouse-dsn="tcp://127.0.0.1:${clickhouse_port}" --timeout=10m
done

cd "${siginsight_root}"
LITEQUERY_CLICKHOUSE_DSN="tcp://127.0.0.1:${clickhouse_port}" \
	go test -tags=integration ./pkg/litequery
