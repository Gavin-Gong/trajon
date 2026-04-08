# Chart Widget Design

**Goal:** Add a new `Chart` navigation entry that lets users choose a symbol and view a TradingView widget-based chart inside the desktop workspace.

**Current Context:**
- The desktop app currently has three sections: `analytics`, `calendar`, and `settings`.
- The existing `analytics` section is still an empty placeholder.
- The app runs inside Tauri and already renders a desktop-style React surface.
- The current Tauri config has `csp: null`, which avoids immediate CSP blocking for third-party widget scripts.

## Requirements

1. Add a new main navigation item named `Chart`.
2. Render a dedicated chart page when users select that menu item.
3. Let users choose a symbol from a predefined list.
4. Use a TradingView widget for the first version.
5. Let TradingView provide built-in indicators and drawing tools.
6. Do not build custom chart rendering, custom indicator engines, or drawing persistence in this version.

## Chosen Approach

Use a TradingView widget embedded inside a dedicated `ChartSection` component.

The app owns:
- navigation
- selected symbol
- selected interval
- layout and chart container lifecycle

TradingView owns:
- chart rendering
- indicator controls
- built-in drawing interactions

## Why This Approach

### Recommended approach

Embed a TradingView widget instead of building a custom chart.

Why:
- fastest route to a usable chart page
- indicators and drawing tools are already implemented
- minimal product risk for the first version
- fits the current desktop shell without forcing a new chart architecture

### Alternatives considered

1. Build a fake-data chart page with self-rendered candles and overlays
- Rejected because it increases implementation cost significantly.
- Drawing and indicator behavior would need custom interaction design immediately.

2. Integrate a chart library and custom overlays
- Rejected for now because the product goal is to get a functional chart page quickly.
- Still more complex than embedding TradingView widgets.

## Navigation Design

Add a new section key: `chart`.

Updated main menu:
- `analytics`
- `chart`
- `calendar`
- `settings`

The existing `analytics` placeholder stays intact. `chart` becomes a separate page, not a replacement for `analytics`.

## UI Design

The chart page should contain two parts:

1. Top control bar
- symbol selector
- interval selector
- short helper text indicating that indicators and drawings are available inside the TradingView chart

2. Main chart card
- TradingView widget container
- full-width responsive panel within the current desktop content area

## State Model

The first version only needs view-local chart state.

```ts
export type ChartSymbolOption = {
  label: string;
  value: string;
};

export type ChartIntervalOption = "1" | "5" | "15" | "30" | "60" | "240" | "1D";

export type ChartViewState = {
  symbol: string;
  interval: ChartIntervalOption;
};
```

The chart page should start with a small built-in symbol list such as:

```ts
[
  { label: "BTCUSDT", value: "BINANCE:BTCUSDT" },
  { label: "ETHUSDT", value: "BINANCE:ETHUSDT" },
  { label: "NQ", value: "CME_MINI:NQ1!" },
  { label: "ES", value: "CME_MINI:ES1!" },
]
```

## Component Design

Add a dedicated widget wrapper component, for example:

```ts
function TradingViewWidgetChart({
  symbol,
  interval,
}: {
  symbol: string;
  interval: ChartIntervalOption;
})
```

Responsibilities:
- create a stable DOM container for the widget
- load the TradingView widget script if needed
- initialize the widget with the selected `symbol` and `interval`
- destroy or replace the widget cleanly when inputs change

This wrapper should isolate all third-party integration details from the rest of the desktop surface.

## Lifecycle Rules

1. On first chart page render
- create the widget container
- initialize TradingView with the default symbol and interval

2. On symbol change
- reinitialize or update the widget to the new symbol

3. On interval change
- reinitialize or update the widget to the new interval

4. On unmount
- clean up any injected widget instance and avoid duplicate script setup

## Non-Goals

This version will not include:
- custom candle rendering
- virtual OHLC data generation
- persistent drawings in app storage
- app-managed indicator state beyond the chart's own UI
- syncing chart drawings to journals or presets

## Error Handling

1. Widget script load failure
- render a fallback message inside the chart card
- keep the page usable so the user can retry by changing section or reloading the app

2. Unknown symbol
- restrict selection to built-in options only for version one

3. Resize issues
- keep the widget inside a fixed-height responsive card and allow it to fill width

## Testing Scope

Implementation should verify:
1. the new `Chart` menu item renders in the desktop shell
2. selecting `Chart` renders the chart page instead of the placeholder
3. symbol selector renders with the expected default symbol
4. interval selector renders and updates page state
5. the widget container mounts with stable testable markup

Testing should avoid asserting TradingView internals. The app should test its own wrapper and layout, not the third-party chart implementation itself.

## Summary

This design adds a new `Chart` page backed by a TradingView widget so the app can quickly provide a usable charting surface with built-in indicators and drawing tools. The app remains responsible for navigation, symbol selection, interval selection, and widget lifecycle, while TradingView handles the chart experience itself.
