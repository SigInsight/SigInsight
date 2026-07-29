from datetime import datetime, timedelta, timezone
from http import HTTPStatus
from typing import Callable, List
from uuid import uuid4

import requests

from fixtures import types
from fixtures.auth import USER_ADMIN_EMAIL, USER_ADMIN_PASSWORD
from fixtures.metrics import Metrics


def _headers(token: str) -> dict[str, str]:
    return {"authorization": f"Bearer {token}"}


def _assert_ok(response: requests.Response) -> None:
    assert response.status_code == HTTPStatus.OK, response.text


def _query_payload(signal: str, start: int, end: int) -> dict:
    return {
        "schemaVersion": "v1",
        "start": start,
        "end": end,
        "requestType": "raw",
        "compositeQuery": {
            "queries": [
                {
                    "type": "builder_query",
                    "spec": {
                        "name": "A",
                        "signal": signal,
                        "disabled": False,
                        "limit": 10,
                        "offset": 0,
                        "aggregations": [{"expression": "count()"}],
                    },
                }
            ]
        },
    }


def test_reduced_collector_schema_authenticated_api_smoke(
    signoz: types.SigNoz,
    create_user_admin: None,  # pylint: disable=unused-argument
    get_token: Callable[[str, str], str],
    insert_metrics: Callable[[List[Metrics]], None],
) -> None:
    """Exercise query APIs against the schema produced by the reduced Collector."""
    now = datetime.now(tz=timezone.utc)
    start = int((now - timedelta(minutes=5)).timestamp() * 1000)
    end = int((now + timedelta(minutes=1)).timestamp() * 1000)
    metric_name = "compat.reduced.schema.gauge"
    insert_metrics(
        [
            Metrics(
                metric_name=metric_name,
                labels={"service": "compat-api", "region": "test"},
                resource_attributes={"service.name": "compat-api"},
                timestamp=now,
                value=42.0,
                type_="Gauge",
            )
        ]
    )
    token = get_token(USER_ADMIN_EMAIL, USER_ADMIN_PASSWORD)
    headers = _headers(token)
    api = signoz.self.host_configs["8080"]

    _assert_ok(
        requests.post(
            api.get("/api/v5/query_range"),
            headers=headers,
            json=_query_payload("logs", start, end),
            timeout=10,
        )
    )
    _assert_ok(
        requests.post(
            api.get("/api/v5/query_range"),
            headers=headers,
            json=_query_payload("traces", start, end),
            timeout=10,
        )
    )

    _assert_ok(
        requests.get(
            api.get("/api/v5/metrics/filters/keys"),
            headers=headers,
            params={"searchText": "service", "limit": 10},
            timeout=10,
        )
    )
    _assert_ok(
        requests.post(
            api.get("/api/v5/metrics/filters/values"),
            headers=headers,
            json={"filterKey": "metric_unit", "searchText": "", "limit": 10},
            timeout=10,
        )
    )
    _assert_ok(
        requests.post(
            api.get("/api/v5/metrics/related"),
            headers=headers,
            json={
                "currentMetricName": metric_name,
                "start": start,
                "end": end,
                "filters": {"op": "AND", "items": []},
            },
            timeout=10,
        )
    )
    _assert_ok(
        requests.post(
            api.get("/api/v5/metrics/inspect"),
            headers=headers,
            json={
                "metricName": metric_name,
                "start": start,
                "end": end,
                "filters": {"op": "AND", "items": []},
            },
            timeout=10,
        )
    )

    time_range = {"start": str(start * 1_000_000), "end": str(end * 1_000_000)}
    _assert_ok(
        requests.post(
            api.get("/api/v5/services"), headers=headers, json=time_range, timeout=10
        )
    )
    _assert_ok(
        requests.post(
            api.get("/api/v5/services/dependency_graph"),
            headers=headers,
            json=time_range,
            timeout=10,
        )
    )

    exception_params = {"start": str(start * 1_000_000), "end": str(end * 1_000_000)}
    _assert_ok(
        requests.post(
            api.get("/api/v5/exceptions"),
            headers=headers,
            json=exception_params | {"limit": 10},
            timeout=10,
        )
    )
    _assert_ok(
        requests.post(
            api.get("/api/v5/exceptions/count"),
            headers=headers,
            json=exception_params,
            timeout=10,
        )
    )

    history_params = {
        "start": start,
        "end": end,
        "limit": 10,
        "offset": 0,
        "order": "desc",
    }
    rule_history = f"/api/v5/rules/{uuid4()}/history"
    _assert_ok(
        requests.post(
            api.get(rule_history + "/timeline"),
            headers=headers,
            json=history_params,
            timeout=10,
        )
    )
    _assert_ok(
        requests.post(
            api.get(rule_history + "/top_contributors"),
            headers=headers,
            json=history_params,
            timeout=10,
        )
    )
    _assert_ok(
        requests.post(
            api.get(rule_history + "/overall_status"),
            headers=headers,
            json=history_params,
            timeout=10,
        )
    )

    _assert_ok(
        requests.get(
            api.get("/api/v5/settings/ttl"),
            headers=headers,
            params={"type": "traces"},
            timeout=10,
        )
    )
