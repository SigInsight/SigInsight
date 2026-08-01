import json
import time
import uuid
from datetime import datetime, timedelta, timezone
from typing import Callable, List

from sqlalchemy import text
from wiremock.client import HttpMethods, Mapping, MappingRequest, MappingResponse

from fixtures import types
from fixtures.alertutils import (
    update_rule_channel_name,
    verify_webhook_alert_expectation,
)
from fixtures.metrics import Metrics


def _basic_metric_rule(metric_name: str) -> dict:
    return {
        "alert": "lightweight_metric_threshold",
        "alertType": "METRIC_BASED_ALERT",
        "ruleType": "threshold_rule",
        "condition": {
            "thresholds": {
                "kind": "basic",
                "spec": [
                    {
                        "name": "critical",
                        "target": 10,
                        "matchType": "1",
                        "op": "1",
                        "channels": [],
                    }
                ],
            },
            "compositeQuery": {
                "queryType": "builder",
                "panelType": "graph",
                "queries": [
                    {
                        "type": "builder_query",
                        "spec": {
                            "name": "A",
                            "signal": "metrics",
                            "disabled": False,
                            "stepInterval": "1m",
                            "aggregations": [
                                {
                                    "metricName": metric_name,
                                    "temporality": "unspecified",
                                    "timeAggregation": "latest",
                                    "spaceAggregation": "max",
                                }
                            ],
                        },
                    }
                ],
            },
            "selectedQueryName": "A",
        },
        "evaluation": {
            "kind": "rolling",
            "spec": {"evalWindow": "5m", "frequency": "15s"},
        },
        "labels": {},
        "annotations": {"summary": "lightweight threshold integration"},
        "notificationSettings": {
            "groupBy": [],
            "usePolicy": False,
            "renotify": {"enabled": False, "interval": "30m", "alertStates": []},
        },
        "version": "v5",
        "schemaVersion": "v2alpha1",
    }


def _assert_persisted_route_contains_rule(
    signoz: types.SigNoz, rule_id: str, channel_name: str
) -> None:
    # The test environment deliberately syncs Alertmanager every five seconds.
    # Read the persisted SQLite config after that interval so this covers the
    # route reconstructed from a stored V5 threshold rule, not only its first
    # in-memory evaluation.
    time.sleep(6)
    with signoz.sqlstore.conn.connect() as conn:
        config = conn.execute(
            text(
                "SELECT config FROM alertmanager_config ORDER BY updated_at DESC LIMIT 1"
            )
        ).scalar_one()

    routes = json.loads(config)["route"]["routes"]
    assert any(
        route["receiver"] == channel_name
        and any(rule_id in matcher for matcher in route["matchers"])
        for route in routes
    ), routes


def test_lightweight_metric_threshold_fires(
    signoz: types.SigNoz,
    notification_channel: types.TestContainerDocker,
    make_http_mocks: Callable[[types.TestContainerDocker, List[Mapping]], None],
    create_webhook_notification_channel: Callable[[str, str, dict, bool], str],
    create_alert_rule: Callable[[dict], str],
    insert_metrics: Callable[[List[Metrics]], None],
) -> None:
    """A supported V5 builder rule evaluates Collector-compatible metric data."""
    channel_name = f"lightweight-threshold-{uuid.uuid4()}"
    endpoint_path = f"/alert/{channel_name}"
    endpoint = notification_channel.container_configs["8080"].get(endpoint_path)
    make_http_mocks(
        notification_channel,
        [
            Mapping(
                request=MappingRequest(method=HttpMethods.POST, url=endpoint_path),
                response=MappingResponse(status=200, json_body={}),
                persistent=False,
            )
        ],
    )
    create_webhook_notification_channel(
        channel_name=channel_name,
        webhook_url=endpoint,
        http_config={},
        send_resolved=False,
    )

    now = datetime.now(tz=timezone.utc).replace(second=0, microsecond=0)
    metric_name = "integration.lightweight.threshold.gauge"
    insert_metrics(
        [
            Metrics(
                metric_name=metric_name,
                labels={"service": "integration-api"},
                timestamp=now - timedelta(minutes=2),
                value=12.0,
                temporality="Unspecified",
                type_="Gauge",
            ),
            Metrics(
                metric_name=metric_name,
                labels={"service": "integration-api"},
                timestamp=now - timedelta(minutes=1),
                value=20.0,
                temporality="Unspecified",
                type_="Gauge",
            ),
        ]
    )

    rule = _basic_metric_rule(metric_name)
    update_rule_channel_name(rule, channel_name)
    rule_id = create_alert_rule(rule)
    _assert_persisted_route_contains_rule(signoz, rule_id, channel_name)

    # The rule manager evaluates on a 15-second cadence; allow one cycle plus
    # Alertmanager's configured group wait.
    verify_webhook_alert_expectation(
        notification_channel,
        channel_name,
        types.AlertExpectation(
            should_alert=True,
            wait_time_seconds=30,
            expected_alerts=[
                types.FiringAlert(
                    labels={
                        "alertname": "lightweight_metric_threshold",
                        "threshold.name": "critical",
                    }
                )
            ],
        ),
    )
