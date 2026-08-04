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

The one-line Lite filter DSL is an input projection over `filters.items`, not a
second persisted filter model. A complete expression is parsed with the shared
ANTLR filter grammar, restricted to one flat `AND` or `OR` chain and converted
to typed structured predicates. Only successful parses update both
`filters.items` and the derived `filter.expression`; invalid drafts remain local
to the editor. QuickFilters and row controls update the same structured state,
which is serialized back into the text field.

Composition controls follow the result contract. Time-series/scalar panels can
render multiple builder queries and arithmetic formulas. Raw/trace panels have
one row stream, so duplicate/add-query/add-formula actions are not rendered.

## Consequences

The transition retains the legacy state container temporarily, which limits
M6's deletion count. In return, saved-query compatibility and every existing
panel's V5 serialization are retained. Once M7 demonstrates that supported
pages use the Lite route, M8 can replace the state bridge and remove the
advanced state branches with evidence rather than a bulk DTO rewrite.

The synchronized DSL restores compact keyboard-oriented filter entry without
restoring arbitrary legacy expressions. The tradeoff is that nested boolean
groups cannot be represented by the current single-operator `TagFilter` model
and are rejected before query state changes.
