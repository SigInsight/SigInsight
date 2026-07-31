#!/usr/bin/env bash

set -euo pipefail

# Runs the supported Lite Query capability matrix against real ClickHouse 25.5.6.
# The Collector migration test is deliberately first: it verifies the schema that
# SigInsight reads is created by the current Collector, rather than a hand-built
# test DDL. The query tests then start SigInsight with SQLite and authenticate
# every API request.
siginsight_root="$(git rev-parse --show-toplevel)"
collector_root="${COLLECTOR_ROOT:-$(cd "${siginsight_root}/../OtelCollector" && pwd)}"

if [ ! -f "${collector_root}/Makefile" ]; then
	printf 'Collector repository not found at %s. Set COLLECTOR_ROOT.\n' "${collector_root}" >&2
	exit 1
fi

make -C "${collector_root}" test-migration-integration

cd "${siginsight_root}/tests/integration"
uv run pytest --clickhouse-version=25.5.6 -vv \
	src/compat/01_reduced_schema_api.py \
	src/querier/01_logs.py \
	src/querier/02_logs_json_body.py \
	src/querier/03_metrics.py \
	src/querier/04_traces.py \
	src/querier/05_metrics_rate_cumulative_counter.py \
	src/querier/08_metrics_histogram.py \
	src/querier/09_metrics_gauge.py \
	src/querier/10_metrics_rate_delta_counter.py \
	src/querier/11_cost_meter.py \
	src/alerts/02_basic_alert_conditions.py
