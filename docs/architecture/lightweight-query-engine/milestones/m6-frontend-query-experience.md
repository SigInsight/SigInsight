# M6: Lightweight Frontend Query Experience

Status: Complete

## Objective

Provide a deliberately constrained editor for the V5 query contract. The editor
must construct only the subset accepted by the lightweight query engine while
leaving existing saved queries and the legacy editor available during migration.

## Boundary

The new frontend package owns presentation, local validation, and conversion to
the existing `IBuilderQuery` state model. It does not create a second network
protocol and it does not compile SQL. `prepareQueryRangePayloadV5` remains the
single frontend serializer, and the V5 adapter remains the backend boundary.

```text
Lite query controls
  -> IBuilderQuery / IBuilderFormula (supported fields only)
  -> prepareQueryRangePayloadV5
  -> /api/v5/query_range
  -> lightweight adapter and Lite IR
```

## Capability Rules

The UI derives its option sets from a frontend representation of
`capability-matrix.json` and applies these rules before it writes state:

- one aggregation per builder query;
- Logs and Traces expose count plus their documented typed aggregates;
- Metrics expose Gauge, Sum, Histogram and Meter operations only;
- filters are conjunctions of typed predicates using `=`, `!=`, comparison,
  `IN`, `NOT IN`, `EXISTS`, `NOT EXISTS`, and `CONTAINS`;
- group-by, global order and global limit remain available;
- formulas permit only references joined by `+`, `-`, `*`, and `/`;
- advanced having, post-processing functions, raw SQL, joins, subqueries and
  Trace Operator are absent from the Lite editor.

The backend remains authoritative. Unsupported saved queries never undergo
lossy client-side rewriting: they continue to use the legacy editor and its
controlled backend fallback until M7 migration marks them compatible.

## Rollout

`VITE_LIGHTWEIGHT_QUERY_EDITOR_ENABLED=true` is a build-time opt-in for the
new controls. Operators enable it together with
`SIGNOZ_QUERIER_LIGHTWEIGHT__ENGINE__ENABLED=true`; either flag can be rolled
back independently. M7 will replace this paired deployment configuration with
a server-advertised rollout capability after real Collector readback proves the
route.

## Validation

- unit tests cover capability selection, generated filter expression, formula
  validation and generated V5 payloads;
- component tests cover valid editing and rejected advanced input;
- production build and scoped lint cover the new editor package;
- an existing V5 payload test remains the contract test between the new editor
  state and the HTTP request.

The paired authenticated browser/API run, current Collector Log/Trace readback,
and default-on rollout are intentionally M7 gates. M6 only exposes the editor
when both deployment flags are explicitly enabled.

## Delivered Integration

The `QueryBuilder` entry point now selects the Lite editor for supported state,
so it covers the shared UI used by Logs Explorer, Traces Explorer, Metrics
Explorer, dashboard panels and the alert Query Builder. The Cost Meter has no
interactive query editor; its existing fixed V5 Meter requests are covered by
the M5 authenticated bridge fixture. A saved query containing a Trace Operator,
post-processing function, arbitrary having, unsupported filter, unsupported
panel type or advanced formula remains on the legacy editor.

## Deletion Value

M6 is an extraction that unblocks deletion rather than a parallel query
builder. It concentrates the supported subset behind one small capability
module, so M8 can remove the legacy advanced editor controls, their parser-only
UI paths and related options without changing the V5 request wire format.
