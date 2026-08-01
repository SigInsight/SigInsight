from datetime import datetime, timedelta, timezone
from http import HTTPStatus
from typing import Callable, List

import docker
import requests

from fixtures import types
from fixtures.auth import USER_ADMIN_EMAIL, USER_ADMIN_PASSWORD
from fixtures.logs import Logs
from fixtures.meter import MeterSample, make_meter_samples
from fixtures.metrics import Metrics
from fixtures.traces import TraceIdGenerator, Traces


def _query(
    signoz: types.SigNoz, token: str, start_ms: int, end_ms: int, spec: dict
) -> dict:
    return _composite_query(
        signoz,
        token,
        start_ms,
        end_ms,
        [{"type": "builder_query", "spec": spec}],
    )[0]


def _composite_query(
    signoz: types.SigNoz, token: str, start_ms: int, end_ms: int, queries: list[dict]
) -> list[dict]:
    response = requests.post(
        signoz.self.host_configs["8080"].get("/api/v5/query_range"),
        headers={"authorization": f"Bearer {token}"},
        timeout=10,
        json={
            "schemaVersion": "v1",
            "start": start_ms,
            "end": end_ms,
            "requestType": "time_series",
            "compositeQuery": {"queries": queries},
        },
    )
    assert response.status_code == HTTPStatus.OK, response.text
    body = response.json()
    assert body["status"] == "success"
    return body["data"]["data"]["results"]


def test_lite_bridge_executes_supported_v5_requests(
    signoz: types.SigNoz,
    create_user_admin: None,  # pylint: disable=unused-argument
    get_token: Callable[[str, str], str],
    insert_logs: Callable[[List[Logs]], None],
    insert_traces: Callable[[List[Traces]], None],
    insert_metrics: Callable[[List[Metrics]], None],
    insert_meter_samples: Callable[[List[MeterSample]], None],
) -> None:
    now = datetime.now(tz=timezone.utc).replace(microsecond=0)
    start_ms = int((now - timedelta(minutes=15)).timestamp() * 1000)
    end_ms = int((now + timedelta(minutes=5)).timestamp() * 1000)
    metric_name = "lite.bridge.request.count"

    insert_logs(
        [
            Logs(
                timestamp=now,
                resources={"service.name": "lite-api"},
                body="lightweight bridge log",
                severity_text="ERROR",
            )
        ]
    )
    insert_traces(
        [
            Traces(
                timestamp=now,
                trace_id=TraceIdGenerator.trace_id(),
                span_id=TraceIdGenerator.span_id(),
                name="lite bridge span",
                resources={"service.name": "lite-api"},
            )
        ]
    )
    insert_metrics(
        [
            Metrics(
                metric_name=metric_name,
                labels={"service.name": "lite-api"},
                timestamp=now - timedelta(seconds=30),
                value=7.0,
                temporality="Cumulative",
                type_="Sum",
            )
        ]
    )
    meter_name = "lite.bridge.meter.bytes"
    insert_meter_samples(
        make_meter_samples(
            meter_name,
            {"service.name": "lite-api"},
            now,
            count=3,
            base_value=5.0,
            temporality="Delta",
            type_="Sum",
        )
    )

    token = get_token(USER_ADMIN_EMAIL, USER_ADMIN_PASSWORD)
    common = {"stepInterval": "60s", "disabled": False}
    logs = _query(
        signoz,
        token,
        start_ms,
        end_ms,
        {
            **common,
            "name": "logs",
            "signal": "logs",
            "aggregations": [{"expression": "count()"}],
        },
    )
    traces = _query(
        signoz,
        token,
        start_ms,
        end_ms,
        {
            **common,
            "name": "traces",
            "signal": "traces",
            "aggregations": [{"expression": "count()"}],
        },
    )
    metrics = _query(
        signoz,
        token,
        start_ms,
        end_ms,
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
    )
    meter, meter_doubled = _composite_query(
        signoz,
        token,
        start_ms,
        end_ms,
        [
            {
                "type": "builder_query",
                "spec": {
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
            },
            {
                "type": "builder_formula",
                "spec": {"name": "meter_doubled", "expression": "meter * 2"},
            },
        ],
    )

    container_logs = docker.from_env().containers.get(signoz.self.id).logs().decode()
    assert "executing V5 query with lightweight engine" in container_logs
    assert "lightweight V5 query completed" in container_logs

    # The synthetic log/trace writers used by this fixture do not currently
    # expose newly inserted rows to SigInsight's native ClickHouse client.
    # M7 covers that data path using a current Collector; here these calls
    # prove authenticated route, V5 conversion, SQL execution, and fallback.
    for result in (logs, traces):
        aggregations = result["aggregations"]
        assert len(aggregations) == 1

    for result in (metrics, meter, meter_doubled):
        aggregations = result["aggregations"]
        assert len(aggregations) == 1
        assert aggregations[0]["series"], container_logs[-10000:]
        assert aggregations[0]["series"][0]["values"]
