# ADR-008: Lite Frontend Uses the V5 State Bridge

Status: Accepted

## Context

The frontend already serializes dashboard, Explorer and alert queries through
`prepareQueryRangePayloadV5`. Replacing it while the backend migration is
opt-in would duplicate the public request contract and create two sources of
truth for saved queries.

## Decision

The Lite editor writes only the supported subset of `IBuilderQuery` and
`IBuilderFormula`, then reuses the existing V5 serializer. Its components do
not import legacy Query Builder controls, Trace Operator, Having DSL, function
chain or raw SQL editors. A build-time UI switch controls rollout alongside the
backend engine switch.

## Consequences

The transition retains the legacy state container temporarily, which limits
M6's deletion count. In return, saved-query compatibility and every existing
panel's V5 serialization are retained. Once M7 demonstrates that supported
pages use the Lite route, M8 can replace the state bridge and remove the
advanced state branches with evidence rather than a bulk DTO rewrite.
