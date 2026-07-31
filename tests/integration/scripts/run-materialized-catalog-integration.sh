#!/usr/bin/env bash

set -euo pipefail

siginsight_root="$(git rev-parse --show-toplevel)"
collector_root="${COLLECTOR_ROOT:-$(cd "${siginsight_root}/../OtelCollector" && pwd)}"

if [[ ! -f "${collector_root}/Makefile" ]]; then
	echo "Collector repository not found at ${collector_root}. Set COLLECTOR_ROOT." >&2
	exit 1
fi

make -C "${collector_root}" test-migration-integration

cd "${siginsight_root}/tests/integration"
uv run pytest --clickhouse-version=25.5.6 -vv -s \
	src/compat/04_materialized_catalog.py
