# Chart Persistence Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Persist the chart page's selected symbol and interval through the existing trading panel storage model, without restoring the current main section.

**Architecture:** Extend `TradingPanelState` with a small `chart` object, normalize it when reading local storage, and expose a store action that updates only chart view fields. Replace `ChartSection` local state with values from `useTradingPanelState()` so the UI reads and writes the persisted chart view through the shared store.

**Tech Stack:** React 19, TypeScript, Vitest, Testing Library, localStorage persistence, existing trading panel store

---

### Task 1: Add chart view model to persisted storage

**Files:**
- Modify: `src/model/storage.ts`
- Test: `src/model/storage.test.ts`

**Step 1: Write the failing test**

Extend `src/model/storage.test.ts` to require a persisted chart model on `TradingPanelState`.

```ts
it("includes chart view state in persisted storage models", () => {
  const state: TradingPanelState = {
    journals: [],
    config: { timerEnabled: false, timerInterval: 300 },
    pairPresets: [],
    defaultPresetSymbol: undefined,
    chart: {
      symbol: "BINANCE:BTCUSDT",
      interval: "60",
    },
    panelPinned: true,
  };

  expect(state.chart).toEqual({ symbol: "BINANCE:BTCUSDT", interval: "60" });
});
```

**Step 2: Run test to verify it fails**

Run: `npm test -- src/model/storage.test.ts`
Expected: FAIL because `TradingPanelState` does not yet include `chart`.

**Step 3: Write minimal implementation**

In `src/model/storage.ts`:
- add a `ChartViewState` type
- extend `TradingPanelState` with `chart: ChartViewState`

Use this minimal type:

```ts
export type ChartViewState = {
  symbol: string;
  interval: string;
};
```

**Step 4: Run test to verify it passes**

Run: `npm test -- src/model/storage.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add src/model/storage.ts src/model/storage.test.ts
git commit -m "feat: add chart view storage model"
```

### Task 2: Normalize and update chart state through the store

**Files:**
- Modify: `src/tradingPanelStore.ts`
- Test: `src/tradingPanelStore.test.tsx`

**Step 1: Write the failing test**

Add chart-specific store tests to `src/tradingPanelStore.test.tsx`.

```ts
it("updates and persists chart view state", async () => {
  const { result } = renderHook(() => useTradingPanelState());

  act(() => {
    result.current.actions.updateChartView({
      symbol: "BINANCE:ETHUSDT",
      interval: "240",
    });
  });

  expect(result.current.state.chart).toEqual({
    symbol: "BINANCE:ETHUSDT",
    interval: "240",
  });

  await waitFor(() => {
    expect(window.localStorage.getItem(TRADING_PANEL_STORAGE_KEY)).toContain("BINANCE:ETHUSDT");
  });
});
```

Add a second test showing invalid or missing stored chart values fall back to defaults once the store rehydrates.

**Step 2: Run test to verify it fails**

Run: `npm test -- src/tradingPanelStore.test.tsx`
Expected: FAIL because the store has no `chart` field or `updateChartView` action yet.

**Step 3: Write minimal implementation**

In `src/tradingPanelStore.ts`:
- add `chart` default state:

```ts
chart: {
  symbol: "BINANCE:BTCUSDT",
  interval: "60",
}
```

- define allowed chart symbols and intervals using the same chart option values as the UI
- normalize stored `chart.symbol` and `chart.interval`
- add `updateChartView(next: Partial<ChartViewState>)`
- expose the new action from `useTradingPanelState()`

Normalization rules:
- invalid symbol falls back to default symbol
- invalid interval falls back to default interval
- missing `chart` falls back to default chart view

**Step 4: Run test to verify it passes**

Run: `npm test -- src/tradingPanelStore.test.tsx`
Expected: PASS

**Step 5: Commit**

```bash
git add src/tradingPanelStore.ts src/tradingPanelStore.test.tsx
git commit -m "feat: persist chart view in trading panel store"
```

### Task 3: Replace chart local state with persisted store state

**Files:**
- Modify: `src/tradingPanelApp.tsx`
- Test: `src/tradingPanelApp.test.tsx`

**Step 1: Write the failing test**

Add a UI test that seeds persisted chart state and verifies the chart page restores it.

```ts
it("restores persisted chart symbol and interval when opening the chart page", () => {
  seedState({
    ...buildState(),
    chart: {
      symbol: "BINANCE:ETHUSDT",
      interval: "240",
    },
  });

  render(<TradingPanelDesktop view="main" />);
  fireEvent.click(screen.getByRole("menuitem", { name: /Chart/ }));

  expect(screen.getByTestId("chart-symbol-select")).toHaveValue("BINANCE:ETHUSDT");
  expect(screen.getByTestId("chart-interval-select")).toHaveValue("240");
  expect(screen.getByTestId("tradingview-widget-status")).toHaveTextContent(
    "当前图表: BINANCE:ETHUSDT / 240",
  );
});
```

**Step 2: Run test to verify it fails**

Run: `npm test -- src/tradingPanelApp.test.tsx`
Expected: FAIL because `ChartSection` still uses local `useState` defaults.

**Step 3: Write minimal implementation**

In `src/tradingPanelApp.tsx`:
- pass `store` into `ChartSection`
- replace local `symbol` / `interval` state with `store.state.chart`
- update selectors to call `store.actions.updateChartView(...)`

Do not persist section navigation. The main view should still default to `analytics`.

**Step 4: Run test to verify it passes**

Run: `npm test -- src/tradingPanelApp.test.tsx`
Expected: PASS

**Step 5: Commit**

```bash
git add src/tradingPanelApp.tsx src/tradingPanelApp.test.tsx
git commit -m "feat: restore persisted chart view"
```

### Task 4: Full verification and design sync

**Files:**
- Modify: `docs/plans/2026-04-06-chart-persistence-design.md` (only if wording must match implementation)
- Test: `src/model/storage.test.ts`
- Test: `src/panelWindow.test.ts`
- Test: `src/tradingPanelStore.test.tsx`
- Test: `src/tradingPanelApp.test.tsx`

**Step 1: Run targeted verification**

Run: `npm test -- src/model/storage.test.ts src/panelWindow.test.ts src/tradingPanelStore.test.tsx src/tradingPanelApp.test.tsx`
Expected: PASS

**Step 2: Run build verification**

Run: `npm run build`
Expected: successful TypeScript compile and Vite production build

**Step 3: Reconcile design wording**

If implementation details differ slightly from the design, update `docs/plans/2026-04-06-chart-persistence-design.md` to match reality.

**Step 4: Commit**

```bash
git add docs/plans/2026-04-06-chart-persistence-design.md src/model/storage.ts src/model/storage.test.ts src/tradingPanelStore.ts src/tradingPanelStore.test.tsx src/tradingPanelApp.tsx src/tradingPanelApp.test.tsx
git commit -m "feat: persist chart view state"
```
