#!/usr/bin/env bash

set -euo pipefail

# The Collector migration test writes real OTLP payloads into a local
# ClickHouse 25.5.6 instance. The API test then starts current SigInsight with
# the established ClickHouse/SQLite integration fixtures and authenticates a
# user before exercising every query surface affected by the reduced schema.
siginsight_root="$(git rev-parse --show-toplevel)"
collector_root="${COLLECTOR_ROOT:-$(cd "${siginsight_root}/../OtelCollector" && pwd)}"

make -C "${collector_root}" test-migration-integration

cd "${siginsight_root}/tests/integration"
uv run pytest --clickhouse-version=25.5.6 -vv src/compat/01_reduced_schema_api.py
