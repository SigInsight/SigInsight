# M13: V2 Visualization Boundary and Container Consolidation

Status: Planned
Date: 2026-08-04
Start commit: `47c2003`
Related ADR: None. This milestone changes frontend ownership boundaries only. It does
not change the V5 request/response contract, alert evaluation semantics, or ClickHouse
schema.

## Problem and Goal

The legacy chart implementation has already been removed: there are no production
imports of `components/Uplot` or `lib/uPlotLib`. `uplot` and the `.uplot` styles remain
required because `lib/uPlotV2` is the single active rendering implementation.

The remaining problem is not two chart libraries. It is duplicated orchestration around
the one active chart library:

```text
Explorer / Dashboard / Alert preview
  -> TimeSeriesView or PanelWrapper
      -> PanelVisualization panels
          -> lib/uPlotV2
```

`PanelWrapper` dispatches to `PanelVisualization`, while the latter imports
`PanelWrapper` types, constants, and utilities. `TimeSeriesView` independently prepares
the same aligned data and V2 config as the TimeSeries and Bar panels. The `container`
tree is also being used as a shared library: the current static scan finds 270
container-to-container production imports.

This milestone establishes a one-way visualization boundary and deletes the adapters
that become redundant after migration.

## Scope

- Keep `lib/uPlotV2` as the only chart implementation.
- Give V2 a narrow public API; feature code may not import its internal builder,
  context, tooltip controller, or utility implementation directly.
- Move the four active `uPlotShared` ownership groups to their correct homes, then
  delete `lib/uPlotShared`.
- Move panel dispatch, panel input types, and chart constants under
  `PanelVisualization`, eliminating its dependency on `PanelWrapper`.
- Migrate Dashboard, full view, and Alert preview from `PanelWrapper` to the
  visualization dispatcher, then delete the old dispatcher and mapping.
- Migrate explorer time-series and bar rendering to the same panel data/config path,
  deleting duplicated preparation from `TimeSeriesView`.
- Move cross-feature visualization and drilldown contracts out of feature-private
  `container` paths when that removal is required to break an import cycle.

## Non-goals

- Do not replace the `uplot` package or change chart rendering behavior.
- Do not remove active chart features such as legend visibility, tooltip, selection,
  thresholds, bar/histogram rendering, or dashboard drilldown.
- Do not delete Alert History timeline behavior in this milestone. Its specialized
  plugin may be moved from the V2 core to Alert History, but removing it needs a
  separate product decision.
- Do not perform folder-only moves. Every commit must delete code, remove a proven
  deletion blocker, or add tests that make a planned deletion safe.
- Do not change `/api/v5/query_range`, V5 result models, saved query compatibility, or
  alert rule payload semantics.

## Current Code and Dependencies

Measured on 2026-08-04 from the production tree plus colocated tests:

| Area | Production LOC | Test LOC | Notes |
| --- | ---: | ---: | --- |
| `lib/uPlotV2` | 5,603 | 5,175 | Active chart core; retain |
| `lib/uPlotShared` | 489 total | included | Active helpers with incorrect ownership |
| `PanelVisualization` | 2,945 | 2,061 | Main visualization layer |
| `PanelWrapper` | 929 | 894 | Legacy dispatcher plus retained non-chart panels |
| `TimeSeriesView` | 330 | 0 | Explorer adapter duplicating panel setup |
| `container` total | - | - | 108,278 LOC; 270 production cross-container imports |

The direct dependency cycle to remove first is:

```text
PanelWrapper -> PanelVisualization panels
PanelVisualization -> PanelWrapper types/constants/utils
```

The retained `PanelWrapper` subpanels for table, list, pie, and value are not assumed
dead. They move only after their caller and rendering contracts are covered.

## Design

### 1. V2 Public Boundary

Create a public V2 entry point for the stable rendering surface:

- chart host/wrapper;
- public configuration and legend/interaction types;
- supported plugin registration points.

`UPlotConfigBuilder`, `PlotContext`, tooltip controller, and low-level V2 utilities stay
internal. Specialized views must use a feature-level adapter that returns the public
configuration, rather than importing V2 internals. This keeps chart lifecycle behavior
in one place while allowing Alert History and Metrics Inspect to keep their distinct
data semantics.

### 2. Shared Helper Ownership

Move helpers before deleting `uPlotShared`:

| Existing helper | Target owner | Reason |
| --- | --- | --- |
| `generateColor` | `lib/color` | Trace, span, metric, and chart code use it |
| axis/grid/click/focused-series helpers | `lib/uPlotV2` | Depend on chart behavior |
| `getUPlotChartData` | `PanelVisualization` data adapter | Converts panel response data |

The old directory is deleted only when an import scan finds no `lib/uPlotShared` import,
including tests and mocks.

### 3. Panel Input and Dispatcher

Replace the current all-purpose `PanelWrapperProps` bag with two explicit contracts:

```ts
type VisualizationRequest = {
  widget: Widgets;
  queryResponse: QueryResponse;
  panelType: PANEL_TYPES;
  mode: PanelMode;
};

type VisualizationInteractions = {
  onPointClick?: ...;
  onRangeSelect?: ...;
  onLegendChange?: ...;
  onDrilldown?: ...;
};
```

Panel-specific fields remain local to table/list/value/pie implementations rather than
being forwarded through every chart component. The dispatcher belongs to
`PanelVisualization`; it owns the mapping from panel type to TimeSeries, Bar,
Histogram, Table, List, Value, and Pie renderers. Unsupported panel types return the
existing empty state deliberately, with a direct test.

To break the cycle, move these dependencies into `PanelVisualization` or a neutral
shared module before migrating callers:

- `PanelWrapperProps` and panel mode/type contracts;
- histogram bucket and null-value constants;
- generic chart click payload types;
- generic drilldown interfaces.

`QueryTable` may retain table rendering. It must expose a focused drilldown API instead
of being imported by chart hooks through a feature-private path.

### 4. Explorer Chart Adapter

Extract explorer-only concerns from `TimeSeriesView`:

- URL/global-time synchronization becomes `useExplorerTimeRangeSelection`;
- loading, error, and empty state selection stays in the Explorer feature;
- V5 response to aligned data and V2 config uses the same panel adapters as Dashboard;
- chart rendering calls the visualization dispatcher with
  `PanelMode.STANDALONE_VIEW`.

After Logs, Traces, and Metrics Explorer use this path, delete duplicated calls to
`prepareChartData`, `prepareUPlotConfig`, `prepareBarPanelData`, and
`prepareBarPanelConfig` from `TimeSeriesView`.

### 5. Specialized Charts

Alert History timeline and statistics, and Metrics Inspect currently construct V2
configuration directly. They migrate to feature adapters over the public V2 boundary.
The Alert History timeline plugin is moved beside Alert History because it is a
domain-specific renderer, not a reusable V2 primitive.

Removing the timeline plugin is intentionally excluded. It would simplify the code but
would reduce Alert History UI capability, so it requires an explicit product decision.

## API, IR, and Schema Changes

- `/api/v5/query_range`, its request/response DTOs, pagination and warning semantics do
  not change.
- The lightweight query IR, ClickHouse schema, SQLite schema, and Collector write path
  do not change.
- `VisualizationRequest` and `VisualizationInteractions` are frontend-internal
  contracts replacing the broad `PanelWrapperProps` forwarding surface.
- Saved dashboards and alert rules continue to use the existing `Widgets` and query
  payloads. Conversion happens at the visualization boundary and is not persisted.

## Migration and Rollback

1. Add tests around the existing dispatcher and extract neutral contracts without
   behavior changes.
2. Migrate one caller class at a time: dashboard card, full view, Alert preview, then
   explorers.
3. After each caller class, remove its old adapter import and run a production import
   scan.
4. Delete `PanelWrapper` only when the mapping and dispatcher have zero production,
   dynamic, and test-only compatibility consumers.
5. Keep every commit independently buildable. Rollback is a revert of the current
   milestone commit; no persisted data or API migration is involved.

## Test Plan

- Unit tests for panel-type dispatch, panel input conversion, and unsupported types.
- Preserve and extend V2 tests for config, tooltip, legend, selection, histogram, and
  bar behavior whenever ownership changes.
- Add explorer adapter tests for URL time selection and all three loading/empty states.
- Add direct tests for dashboard card, full view, and Alert preview inputs to the new
  dispatcher.
- Production-import scans:

```bash
rg -n "components/Uplot|components/UPlot|lib/uPlotLib" frontend/src
rg -n "lib/uPlotShared" frontend/src
rg -n "from 'container/PanelWrapper" frontend/src
```

- Run targeted Jest suites, `yarn --cwd frontend lint`, `yarn --cwd frontend build`,
  and the existing authenticated Playwright checks for Logs, Traces, Metrics,
  Dashboard/full view, Alert preview, and Alert History.

## Acceptance Matrix

| Capability | Required evidence |
| --- | --- |
| Legacy uPlot remains absent | production import scan is empty |
| V2 chart parity | V2 config/tooltip/legend/selection tests and visual regression |
| Dashboard and full view | panel selection, drilldown, loading, empty, and error states |
| Logs/Traces/Metrics explorers | time-series and bar data, range selection, loading/error/empty states |
| Basic alerts | chart preview and threshold preview still render |
| Alert History | timeline and statistics render unchanged unless a later product decision removes them |
| Deletion safety | typecheck, lint, production build, import scan, and relevant Playwright flow |

## Implementation Results

Not started. No production refactor is permitted until this design is reviewed and the
documentation commit is present.

## Planned Deletions

- `lib/uPlotShared` after helper migration.
- `PanelWrapper/PanelWrapper.tsx` and its dispatch mapping after all retained callers
  use `PanelVisualization`.
- The duplicate chart-data/config branch in `TimeSeriesView` after explorer migration.
- Feature-specific compatibility props, mocks, and styles revealed by each deleted
  adapter.

## Measurement Targets

| Metric | Baseline | Exit condition |
| --- | ---: | --- |
| Legacy uPlot production imports | 0 | remains 0 |
| `lib/uPlotShared` imports | active | 0 and directory deleted |
| `PanelVisualization -> PanelWrapper` imports | active cycle | 0 |
| `PanelVisualization -> QueryTable` private imports | active | 0 or neutral public drilldown API |
| `TimeSeriesView` duplicated chart preparation | 4 helper calls | 0 |
| `container` cross-container production imports | 270 | decreases; no new reverse dependency |
| Milestone production LOC | baseline recorded above | net negative |

Every implementation commit records added/deleted production LOC and the imports it
removed. Test LOC is reported separately and is not counted as production complexity.

## Residual Risks and Follow-up

- `CreateAlertV2` and `MetricsExplorer` are large active features, not automatic
  deletion candidates. Their advanced saved-rule and metric compatibility paths need
  route and persisted-payload evidence before removal.
- The most meaningful optional V2 deletion is the 645-line Alert History timeline
  plugin. It remains until a product decision explicitly accepts a simpler Alert
  History view.
- Container folder moves without a deleted dependency are prohibited by the milestone
  rules because they increase churn without reducing maintenance cost.
