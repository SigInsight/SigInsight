import time
from datetime import datetime, timedelta, timezone
from http import HTTPStatus

import docker
import requests

from fixtures import types
from fixtures.auth import USER_ADMIN_EMAIL, USER_ADMIN_PASSWORD


def _any_value(value: str) -> dict:
    return {"stringValue": value}


def _resource() -> dict:
    return {
        "attributes": [
            {"key": "service.name", "value": _any_value("lite-collector")},
        ]
    }


def _post(endpoint: str, path: str, payload: dict) -> None:
    response = requests.post(f"{endpoint}{path}", json=payload, timeout=10)
    assert response.status_code == HTTPStatus.OK, response.text


def _emit_collector_data(endpoint: str, now: datetime) -> str:
    timestamp = str(int(now.timestamp() * 1_000_000_000))
    # The current Collector OTLP/HTTP receiver accepts trace identifiers as
    # fixed-width hexadecimal strings, matching its JSON decoder contract.
    trace_id = "0123456789abcdef0123456789abcdef"
    span_id = "0123456789abcdef"

    _post(
        endpoint,
        "/v1/logs",
        {
            "resourceLogs": [
                {
                    "resource": _resource(),
                    "scopeLogs": [
                        {
                            "scope": {"name": "lite-collaboration"},
                            "logRecords": [
                                {
                                    "timeUnixNano": timestamp,
                                    "severityText": "ERROR",
                                    "body": _any_value("lightweight collector log"),
                                }
                            ],
                        }
                    ],
                }
            ]
        },
    )
    _post(
        endpoint,
        "/v1/traces",
        {
            "resourceSpans": [
                {
                    "resource": _resource(),
                    "scopeSpans": [
                        {
                            "scope": {"name": "lite-collaboration"},
                            "spans": [
                                {
                                    "traceId": trace_id,
                                    "spanId": span_id,
                                    "name": "lightweight collector span",
                                    "startTimeUnixNano": timestamp,
                                    "endTimeUnixNano": str(int(timestamp) + 1_000_000),
                                }
                            ],
                        }
                    ],
                }
            ]
        },
    )
    _post(
        endpoint,
        "/v1/metrics",
        {
            "resourceMetrics": [
                {
                    "resource": _resource(),
                    "scopeMetrics": [
                        {
                            "scope": {"name": "lite-collaboration"},
                            "metrics": [
                                {
                                    "name": "lite.collector.request.count",
                                    "sum": {
                                        "aggregationTemporality": 2,
                                        "isMonotonic": True,
                                        "dataPoints": [
                                            {"timeUnixNano": timestamp, "asDouble": 7.0}
                                        ],
                                    },
                                }
                            ],
                        }
                    ],
                }
            ]
        },
    )
    return "lite.collector.request.count"


def _query(
    signoz: types.SigNoz, token: str, start_ms: int, end_ms: int, spec: dict
) -> dict:
    response = requests.post(
        signoz.self.host_configs["8080"].get("/api/v5/query_range"),
        headers={"authorization": f"Bearer {token}"},
        timeout=10,
        json={
            "schemaVersion": "v1",
            "start": start_ms,
            "end": end_ms,
            "requestType": "time_series",
            "compositeQuery": {"queries": [{"type": "builder_query", "spec": spec}]},
            "noCache": True,
        },
    )
    assert response.status_code == HTTPStatus.OK, response.text
    body = response.json()
    assert body["status"] == "success"
    return body["data"]["data"]["results"][0]


def _register_admin(signoz: types.SigNoz) -> None:
    response = requests.post(
        signoz.self.host_configs["8080"].get("/api/v5/register"),
        json={
            "name": "lite collector admin",
            "orgName": "lite-collector.integration",
            "email": USER_ADMIN_EMAIL,
            "password": USER_ADMIN_PASSWORD,
        },
        timeout=10,
    )
    assert response.status_code == HTTPStatus.OK, response.text


def _access_token(signoz: types.SigNoz) -> str:
    endpoint = signoz.self.host_configs["8080"]
    context_response = requests.get(
        endpoint.get("/api/v5/sessions/context"),
        params={"email": USER_ADMIN_EMAIL, "ref": endpoint.base()},
        timeout=10,
    )
    assert context_response.status_code == HTTPStatus.OK, context_response.text
    org_id = context_response.json()["data"]["orgs"][0]["id"]

    token_response = requests.post(
        endpoint.get("/api/v5/sessions/email_password"),
        json={
            "email": USER_ADMIN_EMAIL,
            "password": USER_ADMIN_PASSWORD,
            "orgId": org_id,
        },
        timeout=10,
    )
    assert token_response.status_code == HTTPStatus.OK, token_response.text
    return token_response.json()["data"]["accessToken"]


def _has_values(result: dict) -> bool:
    aggregations = result.get("aggregations", [])
    return bool(
        aggregations
        and aggregations[0].get("series")
        and aggregations[0]["series"][0].get("values")
    )


def _collector_row_counts(clickhouse: types.TestContainerClickhouse) -> dict[str, int]:
    tables = {
        "logs": "siginsight_logs.logs",
        "traces": "siginsight_traces.spans",
        "metrics": "siginsight_metrics.metric_points",
        "metric_series": "siginsight_metrics.metric_series",
        "meter": "siginsight_meter.meter_points",
    }
    return {
        signal: int(
            clickhouse.conn.query(f"SELECT count() FROM {table}").result_rows[0][0]
        )
        for signal, table in tables.items()
    }


def _collector_time_ranges(
    clickhouse: types.TestContainerClickhouse,
) -> dict[str, list]:
    return {
        "logs": clickhouse.conn.query(
            "SELECT min(timestamp), max(timestamp) FROM siginsight_logs.logs"
        ).result_rows[0],
        "traces": clickhouse.conn.query(
            "SELECT min(timestamp), max(timestamp) FROM siginsight_traces.spans"
        ).result_rows[0],
    }


def _collector_meter_name(clickhouse: types.TestContainerClickhouse) -> str:
    """Select an actual Delta Sum emitted by the current Collector."""

    rows = clickhouse.conn.query(
        "SELECT metric_name FROM siginsight_meter.meter_points "
        "WHERE lower(temporality) = 'delta' AND lower(type) = 'sum' "
        "GROUP BY metric_name ORDER BY metric_name ASC LIMIT 1"
    ).result_rows
    assert rows, "Collector did not emit a Delta Sum meter sample"
    return str(rows[0][0])


def _collector_meter_range(
    clickhouse: types.TestContainerClickhouse,
) -> tuple[int, int]:
    """Return a request range containing Collector's hour-rounded meter rows."""

    start, end = clickhouse.conn.query(
        "SELECT min(unix_milli), max(unix_milli) FROM siginsight_meter.meter_points"
    ).result_rows[0]
    assert (
        start is not None and end is not None
    ), "Collector did not timestamp meter meter_points"
    return int(start), int(end) + 60_000


def _direct_lite_log_buckets(
    clickhouse: types.TestContainerClickhouse, start_ms: int, end_ms: int
) -> list:
    """Run the exact lightweight log aggregation through ClickHouse HTTP.

    This keeps the collaboration test diagnostic: a missing direct bucket is a
    compiler/time-range failure, while a missing V5 bucket after this assertion
    is a SigInsight native-driver or response-boundary failure.
    """

    return clickhouse.conn.query(
        "SELECT intDiv(siginsight_logs.logs.timestamp, toUInt64({step_ns:UInt64})) "
        "* toUInt64({step_ms:UInt64}) AS timestamp, count() AS value "
        "FROM siginsight_logs.logs "
        "WHERE siginsight_logs.logs.timestamp >= toUInt64({start_ns:UInt64}) "
        "AND siginsight_logs.logs.timestamp < toUInt64({end_ns:UInt64}) "
        "GROUP BY intDiv(siginsight_logs.logs.timestamp, toUInt64({step_ns:UInt64})) "
        "* toUInt64({step_ms:UInt64}) "
        "ORDER BY timestamp ASC LIMIT {limit:UInt64}",
        parameters={
            "step_ns": 60_000_000_000,
            "step_ms": 60_000,
            "start_ns": start_ms * 1_000_000,
            "end_ns": end_ms * 1_000_000,
            "limit": 100,
        },
    ).result_rows


def _query_log(clickhouse: types.TestContainerClickhouse, table: str) -> list:
    clickhouse.conn.command("SYSTEM FLUSH LOGS")
    return clickhouse.conn.query(
        "SELECT query, exception_code, exception, read_rows, read_bytes "
        "FROM system.query_log "
        "WHERE query LIKE {table:String} "
        "ORDER BY event_time_microseconds DESC LIMIT 5",
        parameters={"table": f"%{table}%"},
    ).result_rows


def _latest_query_log(clickhouse: types.TestContainerClickhouse, table: str) -> dict:
    """Return the most recent successful physical-table statement and scan cost."""

    clickhouse.conn.command("SYSTEM FLUSH LOGS")
    rows = clickhouse.conn.query(
        "SELECT query, exception_code, exception, read_rows, read_bytes "
        "FROM system.query_log "
        "WHERE query LIKE {table:String} "
        "ORDER BY event_time_microseconds DESC LIMIT 1",
        parameters={"table": f"%{table}%"},
    ).result_rows
    assert rows, f"no ClickHouse query-log row found for {table}"
    query, exception_code, exception, read_rows, read_bytes = rows[0]
    assert (
        exception_code == 0
    ), f"query against {table} failed: code={exception_code}, error={exception}, sql={query}"
    return {
        "read_bytes": int(read_bytes),
        "read_rows": int(read_rows),
        "sql": str(query),
    }


def _assert_no_lite_query_errors(clickhouse: types.TestContainerClickhouse) -> None:
    """Ensure the real V5 statements did not hide a schema/SQL error."""

    clickhouse.conn.command("SYSTEM FLUSH LOGS")
    failures = clickhouse.conn.query(
        "SELECT query, exception_code, exception FROM system.query_log "
        "WHERE exception_code != 0 AND ("
        "query LIKE '%FROM siginsight_logs.logs%' "
        "OR query LIKE '%FROM siginsight_traces.spans%' "
        "OR query LIKE '%FROM siginsight_metrics.metric_points%' "
        "OR query LIKE '%FROM siginsight_meter.meter_points%') "
        "ORDER BY event_time_microseconds DESC LIMIT 20"
    ).result_rows
    assert not failures, f"lightweight query emitted ClickHouse errors: {failures}"


def _siginsight_logs(signoz: types.SigNoz) -> str:
    return docker.from_env().containers.get(signoz.self.id).logs().decode()[-8_000:]


def _wait_for_collector_rows(
    clickhouse: types.TestContainerClickhouse,
) -> dict[str, int]:
    counts = _collector_row_counts(clickhouse)
    for _ in range(50):
        counts = _collector_row_counts(clickhouse)
        if all(
            counts[signal] > 0
            for signal in ("logs", "traces", "metrics", "metric_series", "meter")
        ):
            return counts
        time.sleep(0.2)
    return counts


def test_lite_query_reads_data_written_by_current_collector(
    signoz_current_collector: types.SigNoz,
    current_collector: str,
    clickhouse: types.TestContainerClickhouse,
) -> None:
    """Verify the Lite V5 boundary reads data written by the current Collector."""

    now = datetime.now(tz=timezone.utc).replace(microsecond=0)
    start_ms = int((now - timedelta(minutes=5)).timestamp() * 1000)
    end_ms = int((now + timedelta(minutes=5)).timestamp() * 1000)
    metric_name = _emit_collector_data(current_collector, now)
    row_counts = _wait_for_collector_rows(clickhouse)
    assert all(
        row_counts[signal] > 0
        for signal in ("logs", "traces", "metrics", "metric_series", "meter")
    ), f"Collector accepted OTLP but did not persist all signals: {row_counts}"
    direct_log_buckets = _direct_lite_log_buckets(clickhouse, start_ms, end_ms)
    assert direct_log_buckets, (
        "The lightweight log SQL did not produce a direct ClickHouse bucket: "
        f"rows={row_counts}, ranges={_collector_time_ranges(clickhouse)}"
    )
    meter_name = _collector_meter_name(clickhouse)
    meter_start_ms, meter_end_ms = _collector_meter_range(clickhouse)
    _register_admin(signoz_current_collector)
    token = _access_token(signoz_current_collector)
    # Time-series requests deliberately omit `limit`: the lightweight contract
    # does not implement top-series limiting, and the frontend removes its
    # stale raw/trace limit before sending this request type.
    common = {"stepInterval": "60s", "disabled": False}
    specs = [
        {
            **common,
            "name": "logs",
            "signal": "logs",
            "aggregations": [{"expression": "count()"}],
        },
        {
            **common,
            "name": "traces",
            "signal": "traces",
            "aggregations": [{"expression": "count()"}],
        },
        {
            **common,
            "name": "metrics",
            "signal": "metrics",
            "aggregations": [
                {
                    "metricName": metric_name,
                    "temporality": "cumulative",
                    "timeAggregation": "sum",
                    "spaceAggregation": "sum",
                }
            ],
        },
        {
            **common,
            "name": "meter",
            "signal": "metrics",
            "source": "meter",
            "aggregations": [
                {
                    "metricName": meter_name,
                    "temporality": "delta",
                    "timeAggregation": "sum",
                    "spaceAggregation": "sum",
                }
            ],
        },
    ]

    physical_tables = {
        "logs": "siginsight_logs.logs",
        "traces": "siginsight_traces.spans",
        "metrics": "siginsight_metrics.metric_points",
        "meter": "siginsight_meter.meter_points",
    }
    for spec in specs:
        query_start_ms, query_end_ms = start_ms, end_ms
        if spec["name"] == "meter":
            query_start_ms, query_end_ms = meter_start_ms, meter_end_ms
        lite_result = {}
        for _ in range(50):
            lite_result = _query(
                signoz_current_collector, token, query_start_ms, query_end_ms, spec
            )
            if _has_values(lite_result):
                break
            time.sleep(0.2)
        assert _has_values(lite_result), (
            f"Collector data was not visible for Lite {spec['name']}: result={lite_result}, "
            f"rows={row_counts}, ranges={_collector_time_ranges(clickhouse)}, "
            f"query_log={_query_log(clickhouse, 'signoz_' + spec['name'] + '.')}, "
            f"siginsight_logs={_siginsight_logs(signoz_current_collector)}"
        )
        lite_query_log = _latest_query_log(clickhouse, physical_tables[spec["name"]])

        print(
            f"{spec['name']} query-log: "
            f"lite rows={lite_query_log['read_rows']} bytes={lite_query_log['read_bytes']}"
        )
    _assert_no_lite_query_errors(clickhouse)
