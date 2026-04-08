# Chart Persistence Design

**Goal:** Persist the chart page's selected `symbol` and `interval` so the app restores the user's last chart view without restoring the current desktop section.

**Current Context:**
- The `Chart` page now exists inside `src/tradingPanelApp.tsx`.
- `ChartSection` currently keeps `symbol` and `interval` in local component state.
- The rest of the app already persists state through `TradingPanelState` in `tradingPanelStore.ts`.
- The user explicitly wants chart parameters restored, but does not want the app to reopen directly into the `Chart` section.

## Requirements

1. Persist the selected chart `symbol`.
2. Persist the selected chart `interval`.
3. Restore those values on next app load.
4. Do not persist or restore the current main navigation section.
5. Reuse the existing storage model and normalize invalid stored values.

## Chosen Approach

Store chart view state inside the existing `TradingPanelState` model.

```ts
export type ChartViewState = {
  symbol: string;
  interval: string;
};

export type TradingPanelState = {
  journals: TradeJournal[];
  config: GlobalConfig;
  pairPresets: TradingPairPreset[];
  defaultPresetSymbol?: string;
  chart: ChartViewState;
  panelPinned: boolean;
};
```

## Why This Approach

### Recommended approach

Extend the existing persisted panel state instead of creating a new chart-only storage key.

Why:
- the app already has a stable storage read/write path
- normalization and storage event syncing already exist
- chart state is currently only two small fields
- implementation cost stays low and consistent with current architecture

### Alternatives considered

1. Separate chart storage key
- Cleaner separation.
- Rejected for now because it adds extra persistence logic for very little state.

2. Local component-only localStorage usage
- Smallest code change.
- Rejected because it bypasses the app's existing model/store architecture.

## Data Model

Recommended defaults:

```ts
const defaultChartView: ChartViewState = {
  symbol: "BINANCE:BTCUSDT",
  interval: "60",
};
```

Allowed values should align with the chart UI controls:
- symbols from the built-in chart symbol list
- intervals from the built-in chart interval list

## Normalization Rules

When reading persisted state:

1. `chart.symbol`
- must be a string
- must be one of the allowed chart symbols
- otherwise fall back to the default symbol

2. `chart.interval`
- must be a string
- must be one of the allowed chart intervals
- otherwise fall back to the default interval

3. Missing `chart`
- fall back to the default chart view

## Store Changes

Add one chart-specific action to the store:

```ts
updateChartView(next: Partial<ChartViewState>)
```

Behavior:
- updates only `chart.symbol` and/or `chart.interval`
- preserves the rest of `TradingPanelState`
- writes through the same persistence path as the rest of panel state

## UI Changes

`ChartSection` should stop owning local `symbol` and `interval` state.

Instead:
- read `store.state.chart.symbol`
- read `store.state.chart.interval`
- call `store.actions.updateChartView()` from the selectors

This keeps the chart page in sync with persisted state without changing navigation behavior.

## Restore Behavior

On app launch:
- the app still opens on the default desktop section
- if the user later clicks `Chart`, the page should show the last persisted `symbol` and `interval`

Non-goal:
- do not remember whether the app was last on `analytics`, `chart`, `calendar`, or `settings`

## Testing Scope

Implementation should verify:
1. storage normalization restores valid chart state
2. invalid stored chart symbol falls back to default
3. invalid stored chart interval falls back to default
4. chart selectors reflect persisted values after app render
5. changing chart selectors updates local storage through the shared state model

## Summary

This design keeps chart persistence deliberately narrow: it stores only the selected `symbol` and `interval`, restores those values through the existing `TradingPanelState` persistence flow, and leaves main section navigation behavior unchanged.
