# Trading Pair Preset Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add persisted trading-pair presets with a single default preset that auto-prefills only when creating a new journal.

**Architecture:** Keep reminder settings inside `GlobalConfig`, add `pairPresets` and `defaultPresetSymbol` as top-level fields on `TradingPanelState`, and normalize them during storage rehydrate. Extend the settings UI to manage presets and wire the panel's journal-creation flow to prefill the default preset without overriding later manual edits.

**Tech Stack:** React 19, TypeScript, Vitest, Testing Library, Ant Design, Vite localStorage persistence

---

### Task 1: Add preset models to state and storage normalization

**Files:**
- Create: `src/model/tradingPairPreset.ts`
- Modify: `src/model/storage.ts`
- Modify: `src/tradingPanelStore.ts`
- Test: `src/model/storage.test.ts`

**Step 1: Write the failing test**

Update `src/model/storage.test.ts` to assert the new preset model API exists and matches the intended shape.

```ts
import type { TimeframePreset, TradingPairPreset } from "./tradingPairPreset";
import type { TradingPanelState } from "./storage";

it("exposes trading pair preset models through the model layer", () => {
  const timeframe: TimeframePreset = "5m";
  const preset: TradingPairPreset = {
    symbol: "BTCUSDT",
    timeframe,
  };
  const state: TradingPanelState = {
    journals: [],
    config: { timerEnabled: false, timerInterval: 300 },
    pairPresets: [preset],
    defaultPresetSymbol: "BTCUSDT",
    panelPinned: true,
  };

  expect(state.pairPresets[0]).toEqual({ symbol: "BTCUSDT", timeframe: "5m" });
  expect(state.defaultPresetSymbol).toBe("BTCUSDT");
});
```

**Step 2: Run test to verify it fails**

Run: `npm test -- src/model/storage.test.ts`
Expected: FAIL because `tradingPairPreset` or the new `TradingPanelState` fields do not exist yet.

**Step 3: Write minimal implementation**

Create `src/model/tradingPairPreset.ts` with:

```ts
export type TimeframePreset =
  | "1m"
  | "3m"
  | "5m"
  | "15m"
  | "30m"
  | "1h"
  | "4h"
  | "1d";

export type TradingPairPreset = {
  symbol: string;
  timeframe: TimeframePreset;
};
```

Update `src/model/storage.ts` to:
- import `TradingPairPreset`
- extend `TradingPanelState` with:
  - `pairPresets: TradingPairPreset[]`
  - `defaultPresetSymbol?: string`
- re-export `TimeframePreset` and `TradingPairPreset`

Update `src/tradingPanelStore.ts` to:
- include `pairPresets: []` and `defaultPresetSymbol: undefined` in `defaultState`
- extend `normalizeState` to validate and normalize preset rows
- drop invalid preset rows individually
- clear `defaultPresetSymbol` if it does not match a normalized preset symbol

Add these minimal helpers inside `src/tradingPanelStore.ts`:

```ts
const TIMEFRAME_PRESETS = new Set(["1m", "3m", "5m", "15m", "30m", "1h", "4h", "1d"]);

function normalizeSymbol(value: string) {
  return value.trim().toUpperCase();
}
```

And a predicate/normalizer pair that:
- requires `symbol` string
- requires `timeframe` string in the allowed set
- stores normalized uppercase symbol

**Step 4: Run test to verify it passes**

Run: `npm test -- src/model/storage.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add src/model/storage.test.ts src/model/tradingPairPreset.ts src/model/storage.ts src/tradingPanelStore.ts
git commit -m "feat: add trading pair preset storage models"
```

### Task 2: Add store actions for preset management

**Files:**
- Modify: `src/tradingPanelStore.ts`
- Test: `src/tradingPanelApp.test.tsx`

**Step 1: Write the failing test**

Add tests in `src/tradingPanelApp.test.tsx` that exercise preset persistence through the rendered app.

Use storage seeding assertions such as:

```ts
it("restores stored pair presets and clears invalid default preset targets", () => {
  seedState({
    journals: [],
    config: { timerEnabled: false, timerInterval: 300 },
    pairPresets: [{ symbol: " btcusdt ", timeframe: "5m" }],
    defaultPresetSymbol: "ETHUSDT",
    panelPinned: true,
  } as TradingPanelState);

  render(<TradingPanelDesktop view="main" />);

  const stored = window.localStorage.getItem(TRADING_PANEL_STORAGE_KEY);
  expect(stored).not.toBeNull();
  expect(stored).toContain("BTCUSDT");
  expect(stored).not.toContain("ETHUSDT");
});
```

Add another test to prove duplicate-symbol save should become update behavior later once the settings UI is wired.

**Step 2: Run test to verify it fails**

Run: `npm test -- src/tradingPanelApp.test.tsx`
Expected: FAIL because the state shape and store logic do not yet fully support the asserted normalization or update behavior.

**Step 3: Write minimal implementation**

In `src/tradingPanelStore.ts`, add new actions:

```ts
upsertPairPreset(symbol: string, timeframe: TimeframePreset)
removePairPreset(symbol: string)
setDefaultPresetSymbol(symbol?: string)
```

Implementation requirements:
- normalize symbol with trim + uppercase
- `upsertPairPreset` updates existing preset by normalized symbol or appends a new one
- `removePairPreset` removes the preset and clears `defaultPresetSymbol` if it matches
- `setDefaultPresetSymbol` stores the normalized symbol only if it exists in `pairPresets`, otherwise clears it

Expose the new actions from `useTradingPanelState()`.

**Step 4: Run test to verify it passes**

Run: `npm test -- src/tradingPanelApp.test.tsx`
Expected: PASS for the new normalization/state behavior assertions.

**Step 5: Commit**

```bash
git add src/tradingPanelApp.test.tsx src/tradingPanelStore.ts
git commit -m "feat: add trading pair preset store actions"
```

### Task 3: Add preset management UI to settings

**Files:**
- Modify: `src/components/tradingPanel/TradingPanelMain.tsx`
- Possibly modify: `src/components/tradingPanel/TradingPanelShared.tsx`
- Test: `src/tradingPanelApp.test.tsx`

**Step 1: Write the failing test**

Add settings-focused UI tests in `src/tradingPanelApp.test.tsx`.

Test cases:

```ts
it("renders trading pair preset controls in settings", () => {
  seedState({
    journals: [],
    config: { timerEnabled: false, timerInterval: 300 },
    pairPresets: [{ symbol: "BTCUSDT", timeframe: "5m" }],
    defaultPresetSymbol: "BTCUSDT",
    panelPinned: true,
  });

  render(<TradingPanelDesktop view="main" />);
  screen.getByRole("menuitem", { name: "设置" }).click();

  expect(screen.getByText("交易对预设")).toBeInTheDocument();
  expect(screen.getByText("BTCUSDT")).toBeInTheDocument();
  expect(screen.getByText("5m")).toBeInTheDocument();
  expect(screen.getByText(/默认预设/)).toBeInTheDocument();
});
```

Add one interaction test that:
- enters a symbol
- selects a timeframe
- saves the preset
- marks it as default
- verifies the updated state reached localStorage

**Step 2: Run test to verify it fails**

Run: `npm test -- src/tradingPanelApp.test.tsx`
Expected: FAIL because the settings UI does not yet render or manage preset controls.

**Step 3: Write minimal implementation**

Update `SettingsSection` in `src/components/tradingPanel/TradingPanelMain.tsx` to add a new card for preset management.

Minimum UI:
- `Input` for symbol
- `Select` for timeframe using the same allowed preset values
- save button that calls `store.actions.upsertPairPreset`
- rendered list of saved presets
- per-row button or radio-like action to mark as default via `store.actions.setDefaultPresetSymbol`
- per-row delete button via `store.actions.removePairPreset`

Keep implementation simple:
- local component state for draft symbol/timeframe
- do not extract a separate component unless the file becomes unwieldy

**Step 4: Run test to verify it passes**

Run: `npm test -- src/tradingPanelApp.test.tsx`
Expected: PASS

**Step 5: Commit**

```bash
git add src/components/tradingPanel/TradingPanelMain.tsx src/tradingPanelApp.test.tsx
git commit -m "feat: add trading pair preset settings"
```

### Task 4: Apply default preset during journal creation only

**Files:**
- Modify: `src/components/tradingPanel/TradingPanelPanel.tsx`
- Possibly modify: `src/components/tradingPanel/TradingPanelShared.tsx`
- Test: `src/tradingPanelApp.test.tsx`

**Step 1: Write the failing test**

Add a panel-flow test in `src/tradingPanelApp.test.tsx`:

```ts
it("prefills the default trading pair preset only when starting a new journal", async () => {
  seedState({
    journals: [],
    config: { timerEnabled: false, timerInterval: 300 },
    pairPresets: [{ symbol: "BTCUSDT", timeframe: "5m" }],
    defaultPresetSymbol: "BTCUSDT",
    panelPinned: false,
  });

  render(<TradingPanelDesktop view="panel" />);
  await userEvent.click(screen.getByRole("button", { name: "开始记录今天的交易" }));

  expect(screen.getByDisplayValue("BTCUSDT")).toBeInTheDocument();
  expect(screen.getByText(/5m/)).toBeInTheDocument();
});
```

Also add a test proving existing journals or manual edits are not retroactively overwritten.

**Step 2: Run test to verify it fails**

Run: `npm test -- src/tradingPanelApp.test.tsx`
Expected: FAIL because journal creation currently only accepts manual symbol and market entry.

**Step 3: Write minimal implementation**

Update the start-journal flow in `src/components/tradingPanel/TradingPanelPanel.tsx`:
- resolve the default preset from `store.state.defaultPresetSymbol` and `store.state.pairPresets`
- prefill the start-journal modal's symbol field from the preset
- expose the default timeframe in the creation UI as contextual text or a disabled field for now, unless the existing form already has a timeframe field
- ensure the prefill happens only when opening the create flow and does not keep reapplying after the user edits input

Important constraint:
- If the current journal model does not yet store timeframe, only prefill and display it in the creation flow for now. Do not invent persistence for journal timeframe unless tests and design are updated to require it.

**Step 4: Run test to verify it passes**

Run: `npm test -- src/tradingPanelApp.test.tsx`
Expected: PASS

**Step 5: Commit**

```bash
git add src/components/tradingPanel/TradingPanelPanel.tsx src/tradingPanelApp.test.tsx
git commit -m "feat: apply default trading pair preset on journal creation"
```

### Task 5: Full verification and documentation sync

**Files:**
- Modify: `docs/plans/2026-04-06-trading-pair-preset-design.md` (only if implementation details diverge)
- Test: `src/model/storage.test.ts`
- Test: `src/panelWindow.test.ts`
- Test: `src/tradingPanelApp.test.tsx`

**Step 1: Run targeted test suite**

Run: `npm test -- src/model/storage.test.ts src/panelWindow.test.ts src/tradingPanelApp.test.tsx`
Expected: PASS

**Step 2: Run full project build**

Run: `npm run build`
Expected: successful TypeScript compile and Vite production build

**Step 3: Reconcile plan vs implementation**

If the implementation required a minor adjustment to the design wording, update `docs/plans/2026-04-06-trading-pair-preset-design.md` to match reality.

**Step 4: Commit**

```bash
git add docs/plans/2026-04-06-trading-pair-preset-design.md src/model/storage.test.ts src/model/tradingPairPreset.ts src/model/storage.ts src/tradingPanelStore.ts src/components/tradingPanel/TradingPanelMain.tsx src/components/tradingPanel/TradingPanelPanel.tsx src/tradingPanelApp.test.tsx
git commit -m "feat: add trading pair presets"
```
