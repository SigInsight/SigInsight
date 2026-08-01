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
# compat starts a second SigInsight instance for the current-Collector path.
# Run it in a separate pytest process so its Alertmanager sync does not share
# SQLite state with the alert webhook suite, matching the CI matrix isolation.
uv run pytest --clickhouse-version=25.5.6 -vv src/compat
uv run pytest --clickhouse-version=25.5.6 -vv src/alerts
