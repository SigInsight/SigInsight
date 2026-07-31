from datetime import datetime, timedelta, timezone
from http import HTTPStatus
from typing import Callable, List

from fixtures import types
from fixtures.auth import USER_ADMIN_EMAIL, USER_ADMIN_PASSWORD
from fixtures.querier import make_query_request
from fixtures.traces import TraceIdGenerator, Traces, TracesKind, TracesStatusCode


def test_trace_materialized_catalog_uses_all_manifest_columns(
    signoz: types.SigNoz,
    create_user_admin: None,  # pylint: disable=unused-argument
    get_token: Callable[[str, str], str],
    insert_traces: Callable[[List[Traces]], None],
    clickhouse: types.TestContainerClickhouse,
) -> None:
    """The V5 boundary must use every column in the static Trace manifest."""

    now = datetime.now(tz=timezone.utc).replace(microsecond=0)
    insert_traces(
        [
            Traces(
                timestamp=now - timedelta(seconds=1),
                duration=timedelta(milliseconds=20),
                trace_id=TraceIdGenerator.trace_id(),
                span_id=TraceIdGenerator.span_id(),
                parent_span_id="",
                name="materialized catalog span",
                kind=TracesKind.SPAN_KIND_SERVER,
                status_code=TracesStatusCode.STATUS_CODE_OK,
                status_message="",
                resources={"service.name": "materialized-catalog-api"},
                attributes={
                    "http.route": "/catalog",
                    "messaging.system": "kafka",
                    "messaging.operation": "publish",
                    "db.system": "postgresql",
                    "rpc.system": "grpc",
                    "rpc.service": "catalog.Catalog",
                    "rpc.method": "Get",
                    "peer.service": "database",
                },
            )
        ]
    )

    token = get_token(USER_ADMIN_EMAIL, USER_ADMIN_PASSWORD)
    queries = [
        {
            "type": "builder_query",
            "spec": {
                "name": field.replace(".", "_"),
                "signal": "traces",
                "stepInterval": 60,
                "disabled": False,
                "groupBy": [
                    {
                        "name": field,
                        "fieldDataType": "string",
                        "fieldContext": context,
                    }
                ],
                "aggregations": [{"expression": "count()"}],
            },
        }
        for field, context in [
            ("service.name", "resource"),
            ("http.route", "attribute"),
            ("messaging.system", "attribute"),
            ("messaging.operation", "attribute"),
            ("db.system", "attribute"),
            ("rpc.system", "attribute"),
            ("rpc.service", "attribute"),
            ("rpc.method", "attribute"),
            ("peer.service", "attribute"),
        ]
    ]
    for request_queries in (queries[:8], queries[8:]):
        response = make_query_request(
            signoz,
            token,
            start_ms=int((now - timedelta(minutes=1)).timestamp() * 1000),
            end_ms=int((now + timedelta(minutes=1)).timestamp() * 1000),
            queries=request_queries,
        )
        assert response.status_code == HTTPStatus.OK, response.text
        assert response.json()["status"] == "success"

    clickhouse.conn.command("SYSTEM FLUSH LOGS")
    manifest_columns = [
        "resource_string_service$$name",
        "attribute_string_http$$route",
        "attribute_string_messaging$$system",
        "attribute_string_messaging$$operation",
        "attribute_string_db$$system",
        "attribute_string_rpc$$system",
        "attribute_string_rpc$$service",
        "attribute_string_rpc$$method",
        "attribute_string_peer$$service",
    ]
    for column in manifest_columns:
        queries = clickhouse.conn.query(
            "SELECT query FROM system.query_log "
            "WHERE query LIKE '%FROM signoz_traces.signoz_index_v3%' "
            "AND query LIKE {column:String} "
            "ORDER BY event_time_microseconds DESC LIMIT 1",
            parameters={"column": f"%{column}%"},
        ).result_rows
        assert queries, f"Lite V5 query did not use manifest column {column}"
