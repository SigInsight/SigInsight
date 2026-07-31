from datetime import datetime, timedelta, timezone
from http import HTTPStatus
from typing import Callable, List

from fixtures import types
from fixtures.auth import USER_ADMIN_EMAIL, USER_ADMIN_PASSWORD
from fixtures.querier import make_query_request
from fixtures.traces import TraceIdGenerator, Traces, TracesKind, TracesStatusCode


def test_trace_materialized_catalog_uses_value_and_exists_columns(
    signoz: types.SigNoz,
    create_user_admin: None,  # pylint: disable=unused-argument
    get_token: Callable[[str, str], str],
    insert_traces: Callable[[List[Traces]], None],
    clickhouse: types.TestContainerClickhouse,
) -> None:
    """The V5 boundary must use the static Trace materialized-field manifest."""

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
                attributes={"http.route": "/catalog"},
            )
        ]
    )

    token = get_token(USER_ADMIN_EMAIL, USER_ADMIN_PASSWORD)
    response = make_query_request(
        signoz,
        token,
        start_ms=int((now - timedelta(minutes=1)).timestamp() * 1000),
        end_ms=int((now + timedelta(minutes=1)).timestamp() * 1000),
        queries=[
            {
                "type": "builder_query",
                "spec": {
                    "name": "traces",
                    "signal": "traces",
                    "stepInterval": 60,
                    "disabled": False,
                        "filter": {
                            "expression": "resource.service.name = 'materialized-catalog-api'"
                        },
                    "groupBy": [
                        {
                            "name": "http.route",
                            "fieldDataType": "string",
                            "fieldContext": "attribute",
                        }
                    ],
                    "aggregations": [{"expression": "count()"}],
                },
            }
        ],
    )
    assert response.status_code == HTTPStatus.OK, response.text
    assert response.json()["status"] == "success"

    clickhouse.conn.command("SYSTEM FLUSH LOGS")
    queries = clickhouse.conn.query(
        "SELECT query FROM system.query_log "
        "WHERE query LIKE '%FROM signoz_traces.signoz_index_v3%' "
        "AND query LIKE '%resource_string_service$$name%' "
        "AND query LIKE '%attribute_string_http$$route%' "
        "ORDER BY event_time_microseconds DESC LIMIT 1"
    ).result_rows
    assert queries, "Lite V5 query did not use the materialized Trace catalog columns"
