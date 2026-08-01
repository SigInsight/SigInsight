#!/usr/bin/env bash

set -euo pipefail

siginsight_root="$(git rev-parse --show-toplevel)"
collector_root="${SIGINSIGHT_OTEL_COLLECTOR_ROOT:-/home/cbw/code/OtelCollector}"

if [[ ! -f "${collector_root}/cmd/siginsightotelcollector/main.go" ]]; then
	echo "SIGINSIGHT_OTEL_COLLECTOR_ROOT must point to the current OtelCollector checkout" >&2
	exit 2
fi

# This test compiles the current Collector source, executes its migrations,
# sends OTLP/HTTP telemetry to it, then queries the current SigInsight branch.
# The Python fixture owns and cleans up its ClickHouse 25.5.6 and SQLite data.
cd "${siginsight_root}/tests/integration"
SIGINSIGHT_OTEL_COLLECTOR_ROOT="${collector_root}" \
	uv run pytest --clickhouse-version=25.5.6 -vv -s \
		src/compat/03_litequery_collector_collaboration.py
