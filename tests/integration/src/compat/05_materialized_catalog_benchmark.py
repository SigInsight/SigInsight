import os
import statistics
import time
import uuid

import pytest

from fixtures import types

BENCHMARK_ROWS_ENV = "M9_MATERIALIZED_BENCHMARK_ROWS"


def _percentile(values: list[float], percentile: float) -> float:
    ordered = sorted(values)
    index = (len(ordered) - 1) * percentile
    lower = int(index)
    upper = min(lower + 1, len(ordered) - 1)
    return ordered[lower] + (ordered[upper] - ordered[lower]) * (index - lower)


def _measure(
    clickhouse: types.TestContainerClickhouse, sql: str, samples: int
) -> list[float]:
    for _ in range(3):
        clickhouse.conn.query(sql)

    timings = []
    for _ in range(samples):
        started = time.perf_counter()
        result = clickhouse.conn.query(sql).result_rows
        timings.append((time.perf_counter() - started) * 1_000)
        assert (
            len(result) == 1 and int(result[0][0]) == 1
        ), f"unexpected benchmark result: {result}"
    return timings


@pytest.mark.skipif(
    os.getenv(BENCHMARK_ROWS_ENV) is None,
    reason=f"set {BENCHMARK_ROWS_ENV} to run the materialized-column benchmark",
)
def test_materialized_trace_catalog_benchmark(
    clickhouse: types.TestContainerClickhouse,
    migrator: types.Operation,  # pylint: disable=unused-argument
) -> None:
    """Emit a local Map-vs-materialized measurement when explicitly requested.

    This is intentionally opt-in: the generated distribution is diagnostic, not
    a proxy for a production cardinality or partition layout.
    """

    rows = int(os.environ[BENCHMARK_ROWS_ENV])
    assert (
        10_000 <= rows <= 1_000_000
    ), "benchmark rows must be between 10000 and 1000000"

    run_id = uuid.uuid4().hex
    table = "siginsight_traces.span_index_v3"
    clickhouse.conn.command(f"TRUNCATE TABLE {table}")
    try:
        clickhouse.conn.command(
            "INSERT INTO siginsight_traces.span_index_v3 "
            "(timestamp, resources_string, attributes_string) "
            "SELECT now64(9), map('service.name', 'benchmark-api'), "
            "map('http.route', if(number % 20 = 0, '/checkout', '/health')) "
            f"FROM numbers({rows})"
        )

        map_sql = (
            f"SELECT count() = {rows // 20} /* m9-map-{run_id} */ FROM {table} "
            "WHERE mapContains(resources_string, 'service.name') "
            "AND resources_string['service.name'] = 'benchmark-api' "
            "AND mapContains(attributes_string, 'http.route') "
            "AND attributes_string['http.route'] = '/checkout'"
        )
        materialized_sql = (
            f"SELECT count() = {rows // 20} /* m9-materialized-{run_id} */ FROM {table} "
            "WHERE `resource_string_service$$name_exists` "
            "AND `resource_string_service$$name` = 'benchmark-api' "
            "AND `attribute_string_http$$route_exists` "
            "AND `attribute_string_http$$route` = '/checkout'"
        )
        map_timings = _measure(clickhouse, map_sql, samples=10)
        materialized_timings = _measure(clickhouse, materialized_sql, samples=10)

        clickhouse.conn.command("SYSTEM FLUSH LOGS")
        logs = clickhouse.conn.query(
            "SELECT query, query_duration_ms, read_rows, read_bytes FROM system.query_log "
            "WHERE query LIKE {run:String} "
            "ORDER BY event_time_microseconds ASC",
            parameters={"run": f"%{run_id}%"},
        ).result_rows
        assert len(logs) >= 20, f"missing benchmark query-log rows: {logs}"

        def report(name: str, timings: list[float]) -> str:
            matching = [row for row in logs if f"m9-{name}-{run_id}" in str(row[0])]
            assert matching, f"missing {name} query-log rows: {logs}"
            return (
                f"{name}: p50={statistics.median(timings):.3f}ms "
                f"p95={_percentile(timings, 0.95):.3f}ms "
                f"read_rows={max(int(row[2]) for row in matching)} "
                f"read_bytes={max(int(row[3]) for row in matching)}"
            )

        print(report("map", map_timings))
        print(report("materialized", materialized_timings))
    finally:
        clickhouse.conn.command(f"TRUNCATE TABLE {table}")
