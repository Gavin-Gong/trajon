# Chart Widget Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a new `Chart` desktop menu with a TradingView widget chart, symbol selector, and interval selector.

**Architecture:** Keep the current desktop shell in `src/tradingPanelApp.tsx`, add a new `chart` section alongside `analytics`, `calendar`, and `settings`, and isolate third-party widget lifecycle inside a dedicated wrapper component. The app controls section state, symbol selection, interval selection, and chart container rendering, while TradingView provides chart rendering, indicators, and drawing tools.

**Tech Stack:** React 19, TypeScript, Ant Design, Vitest, Testing Library, Tauri webview environment, TradingView widget script

---

### Task 1: Add chart section state and desktop navigation entry

**Files:**
- Modify: `src/tradingPanelApp.tsx`
- Test: `src/tradingPanelApp.test.tsx`

**Step 1: Write the failing test**

Add tests in `src/tradingPanelApp.test.tsx` for the new menu entry and page switch.

```ts
it("renders the chart menu entry in the desktop navigation", () => {
  seedState(buildState());
  render(<TradingPanelDesktop view="main" />);

  expect(screen.getByText("Chart")).toBeInTheDocument();
});

it("switches to the chart section when the chart menu item is clicked", () => {
  seedState(buildState());
  render(<TradingPanelDesktop view="main" />);

  fireEvent.click(screen.getByRole("menuitem", { name: /Chart/ }));

  expect(screen.getByText("图表工作台")).toBeInTheDocument();
  expect(screen.queryByText("分析页面暂时留空")).not.toBeInTheDocument();
});
```

**Step 2: Run test to verify it fails**

Run: `npm test -- src/tradingPanelApp.test.tsx`
Expected: FAIL because there is no `Chart` menu item or chart section yet.

**Step 3: Write minimal implementation**

Update `src/tradingPanelApp.tsx` to:
- extend `AppSection` with `chart`
- add a new menu item:

```ts
{
  key: "chart",
  icon: <LineChartOutlined />,
  label: "Chart",
}
```

- render a temporary `ChartSection` when `section === "chart"`

Add a minimal `ChartSection` implementation in the same file that renders:
- title `图表工作台`
- short description
- a placeholder card for the widget area

**Step 4: Run test to verify it passes**

Run: `npm test -- src/tradingPanelApp.test.tsx`
Expected: PASS

**Step 5: Commit**

```bash
git add src/tradingPanelApp.tsx src/tradingPanelApp.test.tsx
git commit -m "feat: add chart navigation section"
```

### Task 2: Add chart state options and controls UI

**Files:**
- Modify: `src/tradingPanelApp.tsx`
- Test: `src/tradingPanelApp.test.tsx`

**Step 1: Write the failing test**

Add tests asserting the chart control bar renders symbol and interval selectors.

```ts
it("renders symbol and interval controls in the chart section", () => {
  seedState(buildState());
  render(<TradingPanelDesktop view="main" />);

  fireEvent.click(screen.getByRole("menuitem", { name: /Chart/ }));

  expect(screen.getByText("选择标的")).toBeInTheDocument();
  expect(screen.getByText("时间周期")).toBeInTheDocument();
  expect(screen.getByText("BINANCE:BTCUSDT")).toBeInTheDocument();
  expect(screen.getByText("60")).toBeInTheDocument();
});
```

**Step 2: Run test to verify it fails**

Run: `npm test -- src/tradingPanelApp.test.tsx`
Expected: FAIL because the chart section does not yet render symbol or interval controls.

**Step 3: Write minimal implementation**

Inside `src/tradingPanelApp.tsx` add:

```ts
const CHART_SYMBOL_OPTIONS = [
  { label: "BTCUSDT", value: "BINANCE:BTCUSDT" },
  { label: "ETHUSDT", value: "BINANCE:ETHUSDT" },
  { label: "NQ", value: "CME_MINI:NQ1!" },
  { label: "ES", value: "CME_MINI:ES1!" },
] as const;

const CHART_INTERVAL_OPTIONS = ["1", "5", "15", "30", "60", "240", "1D"] as const;
```

In `ChartSection`, add local state:
- selected symbol, default `BINANCE:BTCUSDT`
- selected interval, default `60`

Render:
- `Text` label `选择标的`
- `Select` for symbol options
- `Text` label `时间周期`
- `Select` for interval options
- helper copy explaining that indicators and drawings are handled inside the chart widget

**Step 4: Run test to verify it passes**

Run: `npm test -- src/tradingPanelApp.test.tsx`
Expected: PASS

**Step 5: Commit**

```bash
git add src/tradingPanelApp.tsx src/tradingPanelApp.test.tsx
git commit -m "feat: add chart controls"
```

### Task 3: Add a TradingView widget wrapper component

**Files:**
- Modify: `src/tradingPanelApp.tsx`
- Test: `src/tradingPanelApp.test.tsx`

**Step 1: Write the failing test**

Add a test verifying the chart page renders a stable widget mount point.

```ts
it("renders a stable TradingView widget container", () => {
  seedState(buildState());
  render(<TradingPanelDesktop view="main" />);

  fireEvent.click(screen.getByRole("menuitem", { name: /Chart/ }));

  expect(screen.getByTestId("tradingview-widget-shell")).toBeInTheDocument();
  expect(screen.getByTestId("tradingview-widget-container")).toBeInTheDocument();
});
```

**Step 2: Run test to verify it fails**

Run: `npm test -- src/tradingPanelApp.test.tsx`
Expected: FAIL because there is no dedicated widget wrapper or testable container.

**Step 3: Write minimal implementation**

In `src/tradingPanelApp.tsx`, add a dedicated wrapper component:

```ts
function TradingViewWidgetChart({
  symbol,
  interval,
}: {
  symbol: string;
  interval: string;
})
```

Implementation requirements:
- render outer shell with `data-testid="tradingview-widget-shell"`
- render inner mount node with `data-testid="tradingview-widget-container"`
- use `useEffect` and a `ref` to inject or initialize the TradingView widget script only on the client
- keep a clean fallback UI if the script cannot load

For the first pass, it is acceptable to render the stable container plus script bootstrap logic without asserting actual TradingView DOM internals in tests.

**Step 4: Run test to verify it passes**

Run: `npm test -- src/tradingPanelApp.test.tsx`
Expected: PASS

**Step 5: Commit**

```bash
git add src/tradingPanelApp.tsx src/tradingPanelApp.test.tsx
git commit -m "feat: add tradingview widget wrapper"
```

### Task 4: Wire chart controls into widget lifecycle

**Files:**
- Modify: `src/tradingPanelApp.tsx`
- Test: `src/tradingPanelApp.test.tsx`

**Step 1: Write the failing test**

Add a test that proves symbol and interval controls update the wrapper inputs.

Use test-friendly rendered text in the wrapper such as:

```ts
it("updates the chart widget configuration when symbol changes", () => {
  seedState(buildState());
  render(<TradingPanelDesktop view="main" />);

  fireEvent.click(screen.getByRole("menuitem", { name: /Chart/ }));
  fireEvent.mouseDown(screen.getAllByRole("combobox")[0]);
  fireEvent.click(screen.getByText("ETHUSDT"));

  expect(screen.getByText("当前图表: BINANCE:ETHUSDT / 60")).toBeInTheDocument();
});
```

Add a similar test for interval change.

**Step 2: Run test to verify it fails**

Run: `npm test -- src/tradingPanelApp.test.tsx`
Expected: FAIL because the wrapper does not yet expose or react to updated chart config.

**Step 3: Write minimal implementation**

Update `ChartSection` and `TradingViewWidgetChart` so that:
- `ChartSection` passes selected symbol and interval to the wrapper
- the wrapper renders test-friendly status text such as:

```ts
<Text data-testid="tradingview-widget-status">当前图表: {symbol} / {interval}</Text>
```

- the widget `useEffect` depends on `symbol` and `interval`
- on changes, the wrapper reinitializes the widget cleanly instead of stacking multiple script instances into the same container

Keep the implementation minimal. Do not build persistence or a global chart store.

**Step 4: Run test to verify it passes**

Run: `npm test -- src/tradingPanelApp.test.tsx`
Expected: PASS

**Step 5: Commit**

```bash
git add src/tradingPanelApp.tsx src/tradingPanelApp.test.tsx
git commit -m "feat: wire chart controls to tradingview widget"
```

### Task 5: Full verification and doc reconciliation

**Files:**
- Modify: `docs/plans/2026-04-06-chart-widget-design.md` (only if wording needs to match implementation)
- Test: `src/tradingPanelApp.test.tsx`
- Test: `src/tradingPanelStore.test.tsx`
- Test: `src/model/storage.test.ts`
- Test: `src/panelWindow.test.ts`

**Step 1: Run targeted verification suite**

Run: `npm test -- src/model/storage.test.ts src/panelWindow.test.ts src/tradingPanelStore.test.tsx src/tradingPanelApp.test.tsx`
Expected: PASS

**Step 2: Run build verification**

Run: `npm run build`
Expected: successful TypeScript compile and Vite production build

**Step 3: Reconcile design wording**

If implementation details differ slightly from the design, update `docs/plans/2026-04-06-chart-widget-design.md` so the design matches reality.

**Step 4: Commit**

```bash
git add docs/plans/2026-04-06-chart-widget-design.md src/tradingPanelApp.tsx src/tradingPanelApp.test.tsx
git commit -m "feat: add chart widget page"
```
