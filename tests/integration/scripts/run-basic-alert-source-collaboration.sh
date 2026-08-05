#!/usr/bin/env bash

set -euo pipefail

# Exercises the current source backend against the local ClickHouse schema.
# The SQLite database is always temporary and the script deletes its rule.

command -v curl >/dev/null || { echo "curl is required" >&2; exit 1; }
command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }

: "${SIGINSIGHT_TELEMETRYSTORE_CLICKHOUSE_DSN:=tcp://localhost:9000}"
: "${SIGINSIGHT_M15_METRIC_NAME:=http.server.request.duration.sum}"
export SIGINSIGHT_TELEMETRYSTORE_CLICKHOUSE_DSN

verification_dir="$(mktemp -d /tmp/siginsight-alert-collaboration.XXXXXX)"
sqlite_path="${verification_dir}/siginsight.db"
server_log="${verification_dir}/server.log"
server_pid=""
webhook_pid=""
webhook_port_file="${verification_dir}/webhook-port"
webhook_receipt="${verification_dir}/webhook-receipt.json"

cleanup() {
	if [[ -n "${webhook_pid}" ]]; then
		kill "${webhook_pid}" 2>/dev/null || true
		wait "${webhook_pid}" 2>/dev/null || true
	fi
	if [[ -n "${server_pid}" ]]; then
		kill -- "-${server_pid}" 2>/dev/null || kill "${server_pid}" 2>/dev/null || true
		wait "${server_pid}" 2>/dev/null || true
	fi
	rm -rf "${verification_dir}"
}
trap cleanup EXIT INT TERM

export SIGINSIGHT_SQLSTORE_SQLITE_PATH="${sqlite_path}"
export SIGINSIGHT_INSTRUMENTATION_METRICS_READERS_PULL_EXPORTER_PROMETHEUS_PORT=19091
export SIGINSIGHT_PPROF_ENABLED=false
export SIGINSIGHT_WEB_ENABLED=false
export SIGINSIGHT_TOKENIZER_JWT_SECRET="siginsight-alert-collaboration"
export SIGINSIGHT_ALERTMANAGER_PROVIDER=siginsight
# The verifier registers an isolated organization after startup. Keep the
# alertmanager server sync short so its local webhook can be tested too.
export SIGINSIGHT_ALERTMANAGER_SIGINSIGHT_POLL__INTERVAL=1s

# The source server currently listens on 8080. Docker development maps its
# service to host port 8081, leaving 8080 available for this process.
setsid go run -race ./cmd/community server >"${server_log}" 2>&1 &
server_pid=$!

ready=false
for _ in $(seq 1 60); do
	if curl --silent --fail http://127.0.0.1:8080/api/v5/health >/dev/null; then
		ready=true
		break
	fi
	sleep 1
done
if [[ "${ready}" != true ]]; then
	echo "source backend did not become ready" >&2
	tail -n 80 "${server_log}" >&2
	exit 1
fi

base_url=http://127.0.0.1:8080
email=m15-script-verifier@example.invalid
password='M15-script-verifier-password-123'

SIGINSIGHT_M15_WEBHOOK_PORT_FILE="${webhook_port_file}" \
SIGINSIGHT_M15_WEBHOOK_RECEIPT="${webhook_receipt}" \
	node -e '
const fs = require("node:fs");
const http = require("node:http");
const portFile = process.env.SIGINSIGHT_M15_WEBHOOK_PORT_FILE;
const receipt = process.env.SIGINSIGHT_M15_WEBHOOK_RECEIPT;
const server = http.createServer((request, response) => {
  let body = "";
  request.on("data", (chunk) => { body += chunk; });
  request.on("end", () => {
    fs.writeFileSync(receipt, body);
    response.writeHead(200);
    response.end();
  });
});
server.listen(0, "127.0.0.1", () => fs.writeFileSync(portFile, String(server.address().port)));
process.on("SIGTERM", () => server.close(() => process.exit(0)));
' >"${verification_dir}/webhook.log" 2>&1 &
webhook_pid=$!

for _ in $(seq 1 10); do
	if [[ -s "${webhook_port_file}" ]]; then
		break
	fi
	sleep 1
done
if [[ ! -s "${webhook_port_file}" ]]; then
	echo "local webhook did not become ready" >&2
	cat "${verification_dir}/webhook.log" >&2
	exit 1
fi
webhook_url="http://127.0.0.1:$(cat "${webhook_port_file}")/m15"

if ! register_response=$(curl --silent --fail-with-body --show-error --request POST "${base_url}/api/v5/register" \
	--header 'Content-Type: application/json' \
	--data "{\"name\":\"M15 Script Verifier\",\"orgName\":\"m15-script-verification\",\"email\":\"${email}\",\"password\":\"${password}\"}"); then
	echo "registration failed: ${register_response}" >&2
	exit 1
fi

context_response=$(curl --silent --fail --show-error --get "${base_url}/api/v5/sessions/context" \
	--data-urlencode "email=${email}" --data-urlencode "ref=${base_url}")
org_id=$(printf '%s' "${context_response}" | jq -er '.data.orgs[0].id')
token_response=$(curl --silent --fail --show-error --request POST "${base_url}/api/v5/sessions/email_password" \
	--header 'Content-Type: application/json' \
	--data "{\"email\":\"${email}\",\"password\":\"${password}\",\"orgId\":\"${org_id}\"}")
token=$(printf '%s' "${token_response}" | jq -er '.data.accessToken')
auth_header="Authorization: Bearer ${token}"

webhook_channel_name=m15-source-collaboration-webhook
webhook_channel_payload=$(jq -n --arg name "${webhook_channel_name}" --arg url "${webhook_url}" '
{
  name: $name,
  webhook_configs: [{send_resolved: true, url: $url, http_config: {}}]
}')
webhook_channel_response=$(curl --silent --fail --show-error --request POST "${base_url}/api/v5/channels" \
	--header "${auth_header}" --header 'Content-Type: application/json' --data "${webhook_channel_payload}")
printf '%s' "${webhook_channel_response}" | jq -e '.status == "success"' >/dev/null

# SyncServers creates the runtime Alertmanager server for the newly registered
# organization asynchronously. Give the configured one-second poll interval a
# bounded opportunity to observe the new channel and matcher.
sleep 2

end_ms=$(date +%s%3N)
start_ms=$((end_ms - 86400000))

metrics_payload=$(jq -n --argjson start "${start_ms}" --argjson end "${end_ms}" --arg metric "${SIGINSIGHT_M15_METRIC_NAME}" '
{
  schemaVersion: "v1", start: $start, end: $end, requestType: "time_series",
  compositeQuery: {queries: [{type: "builder_query", spec: {
    name: "metrics", signal: "metrics", stepInterval: "60s",
    aggregations: [{metricName: $metric, temporality: "cumulative", timeAggregation: "sum", spaceAggregation: "sum"}],
    filter: {expression: ""}
  }}]}
}')
logs_payload=$(jq -n --argjson start "${start_ms}" --argjson end "${end_ms}" '
{
  schemaVersion: "v1", start: $start, end: $end, requestType: "raw",
  compositeQuery: {queries: [{type: "builder_query", spec: {
    name: "logs", signal: "logs", limit: 5, offset: 0,
    selectFields: [{name: "body", fieldContext: "log"}, {name: "service.name", fieldContext: "resource"}],
    order: [{key: {name: "timestamp", fieldContext: "log"}, direction: "desc"}, {key: {name: "id"}, direction: "desc"}],
    filter: {expression: ""}
  }}]}
}')
traces_payload=$(jq -n --argjson start "${start_ms}" --argjson end "${end_ms}" '
{
  schemaVersion: "v1", start: $start, end: $end, requestType: "raw",
  compositeQuery: {queries: [{type: "builder_query", spec: {
    name: "traces", signal: "traces", limit: 5, offset: 0,
    selectFields: [{name: "name", fieldContext: "span"}, {name: "duration_nano", fieldContext: "span"}],
    order: [{key: {name: "timestamp", fieldContext: "span"}, direction: "desc"}],
    filter: {expression: ""}
  }}]}
}')
exceptions_payload=$(jq -n --argjson start "${start_ms}" --argjson end "${end_ms}" '
{
  schemaVersion: "v1", start: $start, end: $end, requestType: "time_series",
  compositeQuery: {queries: [{type: "builder_query", spec: {
    name: "exceptions", signal: "traces", stepInterval: "60s",
    aggregations: [{expression: "count()", alias: "error_spans"}],
    filter: {expression: "has_error = true"}
  }}]}
}')

post_query() {
	local payload=$1
	local response
	if ! response=$(curl --silent --show-error --fail-with-body --request POST "${base_url}/api/v5/query_range" \
		--header "${auth_header}" --header 'Content-Type: application/json' --data "${payload}"); then
		echo "query_range failed: ${response}" >&2
		tail -n 80 "${server_log}" >&2
		return 1
	fi
	printf '%s' "${response}"
}

metrics_response=$(post_query "${metrics_payload}")
logs_response=$(post_query "${logs_payload}")
traces_response=$(post_query "${traces_payload}")
exceptions_response=$(post_query "${exceptions_payload}")
printf '%s\n' "${metrics_response}" "${logs_response}" "${traces_response}" "${exceptions_response}" | jq -e 'select(.status == "success")' >/dev/null

rule_payload=$(jq -n --arg metric "${SIGINSIGHT_M15_METRIC_NAME}" '{
  schemaVersion: "v3alpha1", alert: "m15 source collaboration rule", alertType: "METRIC_BASED_ALERT",
  ruleType: "threshold_rule", version: "v5",
  condition: {
    kind: "numeric", selectedQueryName: "A", dataQuality: {alertOnNoData: false, minPoints: 1},
    compositeQuery: {queryType: "builder", panelType: "graph", queries: [{type: "builder_query", spec: {
      name: "A", signal: "metrics", stepInterval: "60s",
      aggregations: [{metricName: $metric, temporality: "cumulative", timeAggregation: "sum", spaceAggregation: "sum"}],
      filter: {expression: ""}
    }}]},
    numeric: {reduction: "at_least_once", operator: "gt", thresholds: [{severity: "critical", target: 0, channels: []}]}
  },
  evaluation: {kind: "rolling", spec: {evalWindow: "5m", frequency: "1m"}},
  labels: {verification: "m15"},
  annotations: {description: "source collaboration", summary: "source collaboration"},
  notificationSettings: {groupBy: []}
}')

signal_rule_payload() {
	local alert_type=$1
	local signal=$2
	local filter=$3
	jq -n --arg alert_type "${alert_type}" --arg signal "${signal}" --arg filter "${filter}" '{
  schemaVersion: "v3alpha1", alert: ("m15 " + $signal + " collaboration rule"), alertType: $alert_type,
  ruleType: "threshold_rule", version: "v5",
  condition: {
    kind: "numeric", selectedQueryName: "A", dataQuality: {alertOnNoData: false, minPoints: 1},
    compositeQuery: {queryType: "builder", panelType: "graph", queries: [{type: "builder_query", spec: {
      name: "A", signal: $signal, stepInterval: "60s", aggregations: [{expression: "count()", alias: "event_count"}],
      filter: {expression: $filter}
    }}]},
    numeric: {reduction: "at_least_once", operator: "gt", thresholds: [{severity: "critical", target: 0, channels: []}]}
  },
  evaluation: {kind: "rolling", spec: {evalWindow: "5m", frequency: "1m"}},
  labels: {verification: "m15"}, annotations: {description: "source collaboration", summary: "source collaboration"},
  notificationSettings: {groupBy: []}
}'
}

assert_evaluation_preview() {
	local payload=$1
	local response
	if ! response=$(curl --silent --fail-with-body --show-error --request POST "${base_url}/api/v5/testRule" \
		--header "${auth_header}" --header 'Content-Type: application/json' --data "${payload}"); then
		echo "testRule failed: ${response}" >&2
		tail -n 80 "${server_log}" >&2
		return 1
	fi
	printf '%s' "${response}" | jq -e '.data.evaluationPreview | (.state and (.evaluatedAt > 0))' >/dev/null
}

assert_no_data_preview() {
	local payload=$1
	local response
	if ! response=$(curl --silent --fail-with-body --show-error --request POST "${base_url}/api/v5/testRule" \
		--header "${auth_header}" --header 'Content-Type: application/json' --data "${payload}"); then
		echo "no-data testRule failed: ${response}" >&2
		tail -n 80 "${server_log}" >&2
		return 1
	fi
	printf '%s' "${response}" | jq -e '.data.evaluationPreview | (.alertCount == 1 and .state == "nodata" and (.evaluatedAt > 0))' >/dev/null
}

assert_webhook_delivery() {
	for _ in $(seq 1 15); do
		if [[ -s "${webhook_receipt}" ]] && jq -e '.alerts | length > 0' "${webhook_receipt}" >/dev/null; then
			return
		fi
		sleep 1
	done
	echo "test alert did not reach the local webhook" >&2
	[[ -f "${webhook_receipt}" ]] && cat "${webhook_receipt}" >&2
	exit 1
}

assert_evaluation_preview "${rule_payload}"
assert_evaluation_preview "$(signal_rule_payload 'LOGS_BASED_ALERT' 'logs' '')"
assert_evaluation_preview "$(signal_rule_payload 'TRACES_BASED_ALERT' 'traces' '')"
assert_evaluation_preview "$(signal_rule_payload 'EXCEPTIONS_BASED_ALERT' 'traces' 'has_error = true')"

no_data_rule_payload=$(jq -n \
	--arg channel "${webhook_channel_name}" \
	--arg filter "resource.service.name = 'm15 no data deliberately absent'" '{
  schemaVersion: "v3alpha1", alert: "m15 no data collaboration rule", alertType: "LOGS_BASED_ALERT",
  ruleType: "threshold_rule", version: "v5",
  condition: {
    kind: "numeric", selectedQueryName: "A", dataQuality: {alertOnNoData: true, noDataFor: "30s", minPoints: 1},
    compositeQuery: {queryType: "builder", panelType: "graph", queries: [{type: "builder_query", spec: {
      name: "A", signal: "logs", stepInterval: "60s", aggregations: [{expression: "count()", alias: "event_count"}],
      filter: {expression: $filter}
    }}]},
    numeric: {reduction: "at_least_once", operator: "gt", thresholds: [{severity: "critical", target: 0, channels: [$channel]}]}
  },
  evaluation: {kind: "rolling", spec: {evalWindow: "5m", frequency: "1m"}},
  labels: {verification: "m15"}, annotations: {description: "source collaboration", summary: "source collaboration"},
  notificationSettings: {groupBy: []}
}')
assert_no_data_preview "${no_data_rule_payload}"
assert_webhook_delivery

create_response=$(curl --silent --fail --show-error --request POST "${base_url}/api/v5/rules" \
	--header "${auth_header}" --header 'Content-Type: application/json' --data "${rule_payload}")
rule_id=$(printf '%s' "${create_response}" | jq -er '.data.id')
get_response=$(curl --silent --fail --show-error "${base_url}/api/v5/rules/${rule_id}" --header "${auth_header}")
printf '%s' "${get_response}" | jq -e '.data.schemaVersion == "v3alpha1"' >/dev/null
edited_payload=$(printf '%s' "${rule_payload}" | jq '.alert = "m15 source collaboration edited"')
curl --silent --fail --show-error --request PUT "${base_url}/api/v5/rules/${rule_id}" \
	--header "${auth_header}" --header 'Content-Type: application/json' --data "${edited_payload}" >/dev/null
read_back=$(curl --silent --fail --show-error "${base_url}/api/v5/rules/${rule_id}" --header "${auth_header}")
printf '%s' "${read_back}" | jq -e '.data.alert == "m15 source collaboration edited"' >/dev/null
history_response=$(curl --silent --fail --show-error --request POST "${base_url}/api/v5/rules/${rule_id}/history/timeline" \
	--header "${auth_header}" --header 'Content-Type: application/json' \
	--data "$(jq -n --argjson start "${start_ms}" --argjson end "${end_ms}" '{start: $start, end: $end, offset: 0, limit: 50, order: "desc"}')")
printf '%s' "${history_response}" | jq -e '.data | ((.items | type) == "array" and (.total | type) == "number")' >/dev/null
curl --silent --fail --show-error --request DELETE "${base_url}/api/v5/rules/${rule_id}" --header "${auth_header}" >/dev/null

printf '%s\n' 'basic alert source collaboration passed' \
	'  ClickHouse: configured telemetry store' \
	'  SQLite: fresh temporary database and migrations' \
	'  Query API: metrics, logs, traces, exceptions' \
	'  Alert API: four-signal previews, no-data state, local webhook, history read-back; metrics create, read, edit, delete'
