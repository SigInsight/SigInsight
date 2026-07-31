#!/usr/bin/env bash

set -euo pipefail

# Uses the normal authenticated integration fixture with the Lite bridge
# explicitly enabled. The fixture builds the current SigInsight branch and
# starts ClickHouse 25.5.6 plus SQLite.
siginsight_root="$(git rev-parse --show-toplevel)"

cd "${siginsight_root}/tests/integration"
SIGNOZ_INTEGRATION_LIGHTWEIGHT_ENGINE=true \
	uv run pytest --clickhouse-version=25.5.6 -vv src/compat/02_litequery_v5_bridge.py
