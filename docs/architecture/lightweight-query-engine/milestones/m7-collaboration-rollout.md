# M7: Collaboration Verification and Controlled Rollout

Status: Complete

## Objective

Prove the lightweight engine against data written by the current Collector to
ClickHouse 25.5.6, then make the verified capability discoverable by an
authenticated frontend. This milestone does not delete the legacy engine.

## Design

The existing authenticated `GET /api/v5/features/ui` response is the rollout
authority. It advertises `lightweight_query_engine` only when the running
server has `querier.lightweight_engine_enabled` enabled. The frontend consumes
that response from `AppContext`; a build-time flag must not decide whether it
can send Lite requests to a server.

```text
server config
  -> authenticated feature response
  -> AppContext feature flags
  -> supported QueryBuilder state selects Lite editor
  -> existing V5 serializer and /api/v5/query_range
```

Saved V5 queries are classified without mutation. A query that the existing
`isLiteQueryState` predicate accepts can use the Lite editor when the server
advertises support. A query outside that subset stays in the legacy editor and
uses the existing controlled backend fallback. No saved query is rewritten,
downgraded, or rejected merely because the feature is disabled.

## Verification Plan

The repeatable integration entry point must:

1. start ClickHouse 25.5.6 and SQLite;
2. apply the current Collector schema and receive OTLP Logs, Traces, Metrics,
   and Meter samples through Collector rather than test-side table inserts;
3. start the current SigInsight branch with the lightweight engine enabled;
4. authenticate and call `/api/v5/query_range` for Logs, Traces, Metrics and
   Meter requests;
5. assert that every request returns a time-series value from the data written
   by the current Collector;
6. inspect ClickHouse query logs for statement errors against all four physical
   tables.

`tests/integration/scripts/run-litequery-collector-collaboration.sh` is the
entry point. It requires `SIGNOZ_OTEL_COLLECTOR_ROOT` only when the Collector
checkout is not `/home/cbw/code/OtelCollector`.

Services, exceptions, rule history and retention retain their existing API
verification in the broader integration suite. They are not Lite query-engine
consumers and must not be made contingent on this rollout flag.

## Exit Criteria

- The server-advertised feature is covered by backend and frontend tests.
- The integration script is self-contained and records the tested Collector
  and ClickHouse versions.
- Real Collector write then SigInsight readback succeeds for Logs, Traces,
  Metrics and Meter.
- Query log assertions show no ClickHouse errors against the four Lite tables.
- Default-on Dashboard, Explorer and alert switching remains a later consumer
  migration; advanced saved queries remain on legacy.

## Current Result

The capability negotiation implementation and the repeatable current-Collector
fixture are complete. The fixture passed on ClickHouse 25.5.6 and proved all
of the following:

- current Collector migrations complete;
- OTLP/HTTP Logs, Traces and Metrics are written to their current tables;
- the `signozmeter` connector writes Meter samples;
- authenticated `/api/v5/query_range` reaches the lightweight compiler and
  returns non-empty Log, Trace, Metric and Meter time-series results;
- all corresponding ClickHouse query-log entries finish without errors;
- meter verification uses the Collector's hour-rounded stored timestamp rather
  than the OTLP event timestamp.

During this verification, ClickHouse alias resolution exposed a compiler bug:
a time-series `SELECT ... AS timestamp` made an unqualified `WHERE timestamp`
refer to the millisecond output alias rather than the Log table's nanosecond
column. The compiler now qualifies physical Log and Trace timestamp columns;
exact SQL tests and the real collaboration fixture cover the regression.

The legacy engine and advanced UI remain installed deliberately. M7 verifies
the data path and controlled capability negotiation, not a default-on consumer
migration or full Lite-vs-legacy equivalence matrix.
