# SQLite Trading Panel Persistence Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the trading panel's `localStorage` persistence with a SQLite-backed service layer validated by `zod`, including migrations and journal/event persistence APIs.

**Architecture:** Keep trading rules in `src/tradingPanelDomain.ts`, add `zod` schemas for service inputs and persisted data, add a SQLite repository and service layer, and convert `useTradingPanelState` to async hydration through the service. Tauri runtime uses `@tauri-apps/plugin-sql`; non-Tauri runtime should fail fast, with tests mocking the persistence layer instead of falling back to `localStorage`.

**Tech Stack:** React 19, TypeScript strict, Vitest 3, Tauri 2 plugin-sql, Zod, Bun

---

### Task 1: Add `zod`

**Files:**
- Modify: `package.json`
- Modify: `bun.lock`

**Step 1: Write the failing test**

Create a new test file placeholder in `src/tradingPanelService.test.ts` that imports a schema from `src/tradingPanelSchemas.ts`.

```ts
import { describe, expect, it } from "vitest";
import { tradingPanelStateSchema } from "./tradingPanelSchemas";

describe("tradingPanelSchemas", () => {
  it("parses a persisted trading panel state", () => {
    const state = tradingPanelStateSchema.parse({
      journals: [],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      chart: { symbol: "BINANCE:BTCUSDT", interval: "60" },
      panelPinned: true,
    });

    expect(state.chart.symbol).toBe("BINANCE:BTCUSDT");
  });
});
```

**Step 2: Run test to verify it fails**

Run: `bun run test -- src/tradingPanelService.test.ts`

Expected: FAIL because `src/tradingPanelSchemas.ts` and the `zod` dependency do not exist yet.

**Step 3: Write minimal implementation**

Run: `bun add zod`

**Step 4: Run test to verify it still fails for the right reason**

Run: `bun run test -- src/tradingPanelService.test.ts`

Expected: FAIL because the schema file is still missing, not because the dependency is unresolved.

**Step 5: Commit**

```bash
git add package.json bun.lock src/tradingPanelService.test.ts
git commit -m "build: add zod for persistence validation"
```

### Task 2: Add persisted-data Zod schemas

**Files:**
- Create: `src/tradingPanelSchemas.ts`
- Modify: `src/tradingPanelService.test.ts`

**Step 1: Write the failing test**

Expand `src/tradingPanelService.test.ts` with schema-focused coverage.

```ts
import { describe, expect, it } from "vitest";
import {
  appConfigValueSchema,
  tradeJournalRowSchema,
  tradingPanelStateSchema,
} from "./tradingPanelSchemas";

describe("tradingPanelSchemas", () => {
  it("normalizes symbols in pair presets and default preset", () => {
    const result = tradingPanelStateSchema.parse({
      journals: [],
      config: { timerEnabled: true, timerInterval: 300 },
      pairPresets: [{ symbol: "ethusdt", timeframe: "15m" }],
      defaultPresetSymbol: "ethusdt",
      chart: { symbol: "BINANCE:ETHUSDT", interval: "240" },
      panelPinned: false,
    });

    expect(result.pairPresets).toEqual([{ symbol: "ETHUSDT", timeframe: "15m" }]);
    expect(result.defaultPresetSymbol).toBe("ETHUSDT");
  });

  it("rejects invalid journal rows", () => {
    expect(() =>
      tradeJournalRowSchema.parse({
        id: "journal-1",
        created_at: "2026-04-18T00:00:00.000Z",
        updated_at: "2026-04-18T00:00:00.000Z",
        symbol: "BTCUSDT",
        market: "BINANCE",
        summary: "",
        status: "invalid",
      }),
    ).toThrow();
  });

  it("rejects invalid app config payloads", () => {
    expect(() =>
      appConfigValueSchema.parse({
        chart: { symbol: "BINANCE:BTCUSDT", interval: "60" },
        panelPinned: "yes",
      }),
    ).toThrow();
  });
});
```

**Step 2: Run test to verify it fails**

Run: `bun run test -- src/tradingPanelService.test.ts`

Expected: FAIL because `src/tradingPanelSchemas.ts` does not export these schemas yet.

**Step 3: Write minimal implementation**

In `src/tradingPanelSchemas.ts`, add:
- `tradeJournalStatusSchema`
- `tradeEventTypeSchema`
- `tradeEventSchema`
- `tradeJournalSchema`
- `tradeJournalRowSchema`
- `tradeEventRowSchema`
- `globalConfigSchema`
- `tradingPairPresetSchema` with uppercase symbol normalization
- `chartViewSchema`
- `tradingPanelStateSchema`
- `appConfigValueSchema`

Make schemas perform both validation and normalization where possible.

**Step 4: Run test to verify it passes**

Run: `bun run test -- src/tradingPanelService.test.ts`

Expected: PASS.

**Step 5: Commit**

```bash
git add src/tradingPanelSchemas.ts src/tradingPanelService.test.ts
git commit -m "feat: add zod schemas for trading panel persistence"
```

### Task 3: Add migration runner for trading panel tables

**Files:**
- Create: `src/tradingPanelMigrations.ts`
- Create: `src/tradingPanelRepository.test.ts`

**Step 1: Write the failing test**

Create `src/tradingPanelRepository.test.ts` with a migration test.

```ts
import { describe, expect, it, vi } from "vitest";
import { runTradingPanelMigrations } from "./tradingPanelMigrations";

describe("runTradingPanelMigrations", () => {
  it("creates the base trading panel tables and indexes", async () => {
    const execute = vi.fn().mockResolvedValue(undefined);
    const select = vi.fn().mockResolvedValue([]);
    const db = { execute, select };

    await runTradingPanelMigrations(db as never);

    expect(execute.mock.calls.some(([sql]) => String(sql).includes("CREATE TABLE IF NOT EXISTS trade_journals"))).toBe(true);
    expect(execute.mock.calls.some(([sql]) => String(sql).includes("CREATE TABLE IF NOT EXISTS trade_events"))).toBe(true);
    expect(execute.mock.calls.some(([sql]) => String(sql).includes("CREATE TABLE IF NOT EXISTS app_config"))).toBe(true);
  });
});
```

**Step 2: Run test to verify it fails**

Run: `bun run test -- src/tradingPanelRepository.test.ts`

Expected: FAIL because `src/tradingPanelMigrations.ts` does not exist.

**Step 3: Write minimal implementation**

In `src/tradingPanelMigrations.ts`:
- Define a small `migrations` array with `version` and SQL statements.
- Create `migrations` table if needed.
- Read executed versions.
- Execute the missing migration that creates:
  - `trade_journals`
  - `trade_events`
  - `app_config`
  - required indexes
- Insert the executed migration version.

Use `CREATE TABLE IF NOT EXISTS` and `CREATE INDEX IF NOT EXISTS` for idempotency.

**Step 4: Run test to verify it passes**

Run: `bun run test -- src/tradingPanelRepository.test.ts`

Expected: PASS.

**Step 5: Commit**

```bash
git add src/tradingPanelMigrations.ts src/tradingPanelRepository.test.ts
git commit -m "feat: add sqlite migrations for trading panel persistence"
```

### Task 4: Add SQL repository

**Files:**
- Create: `src/tradingPanelRepository.ts`
- Modify: `src/tradingPanelRepository.test.ts`
- Use: `src/sqliteStorage.ts`
- Use: `src/tradingPanelMigrations.ts`
- Use: `src/tradingPanelSchemas.ts`

**Step 1: Write the failing test**

Expand `src/tradingPanelRepository.test.ts` with repository behavior tests using a mocked SQL database.

Coverage to add:
- `loadState()` returns default state from an empty database.
- `insertTradeJournal()` persists a journal row.
- `appendTradeEvent()` persists an event row and updates the parent journal timestamp.
- `writeAppState()` persists chart/config/panelPinned/default preset values in `app_config`.
- invalid SQL rows are rejected through Zod parsing.

**Step 2: Run test to verify it fails**

Run: `bun run test -- src/tradingPanelRepository.test.ts`

Expected: FAIL because the repository is not implemented.

**Step 3: Write minimal implementation**

Create `src/tradingPanelRepository.ts` with:
- `createTradingPanelRepository()`
- database open through `openTradingFlowDatabase()`
- `runTradingPanelMigrations()` on first use
- `loadState()` to read journals, events, and app config entries
- `insertTradeJournal()`
- `appendTradeEvent()`
- `updateTradeJournal()`
- `writeAppState()`

Use explicit mapping between SQL `snake_case` columns and TypeScript `camelCase` models.

**Step 4: Run test to verify it passes**

Run: `bun run test -- src/tradingPanelRepository.test.ts`

Expected: PASS.

**Step 5: Commit**

```bash
git add src/tradingPanelRepository.ts src/tradingPanelRepository.test.ts
git commit -m "feat: add sql repository for trading panel state"
```

### Task 5: Add trading panel service layer

**Files:**
- Create: `src/tradingPanelService.ts`
- Modify: `src/tradingPanelService.test.ts`
- Use: `src/tradingPanelRepository.ts`
- Use: `src/tradingPanelDomain.ts`
- Use: `src/tradingPanelSchemas.ts`

**Step 1: Write the failing test**

Expand `src/tradingPanelService.test.ts` with service behavior tests using a mocked repository.

Coverage to add:
- `createTradeJournal(symbol, market)` creates a journal through the domain helper and persists it.
- `createTradeEvent(journalId, draft)` rejects invalid event transitions based on current net position/status.
- `createTradeEvent(journalId, draft)` persists successful events and updates journal `updatedAt`.
- `finishTradeJournal(journalId, summary)` trims summary and marks the journal done.
- `updateConfig()` rejects invalid `timerInterval`.
- `setDefaultPresetSymbol()` only accepts a symbol already present in `pairPresets`.

**Step 2: Run test to verify it fails**

Run: `bun run test -- src/tradingPanelService.test.ts`

Expected: FAIL because the service does not exist.

**Step 3: Write minimal implementation**

Create `src/tradingPanelService.ts` with:
- `createTradingPanelService()`
- `loadTradingPanelState()`
- `createTradeJournal()`
- `createTradeEvent()`
- `finishTradeJournal()`
- `updateChartView()`
- `setPanelPinned()`
- `updateConfig()`
- `upsertPairPreset()`
- `removePairPreset()`
- `setDefaultPresetSymbol()`

Keep trading rules in `src/tradingPanelDomain.ts`; the service should orchestrate repository reads/writes and schema parsing only.

**Step 4: Run test to verify it passes**

Run: `bun run test -- src/tradingPanelService.test.ts`

Expected: PASS.

**Step 5: Commit**

```bash
git add src/tradingPanelService.ts src/tradingPanelService.test.ts
git commit -m "feat: add trading panel persistence service"
```

### Task 6: Move `useTradingPanelState` to the service layer

**Files:**
- Modify: `src/tradingPanelStore.ts`
- Modify: `src/tradingPanelStore.test.tsx`

**Step 1: Write the failing test**

Refocus `src/tradingPanelStore.test.tsx` on service-backed behavior.

First tests to add or rewrite:
- the hook hydrates from `loadTradingPanelState()` on mount
- `createJournal()` calls the service and updates local state
- `recordEvent()` calls the service and updates local state
- the hook no longer writes `TRADING_PANEL_STORAGE_KEY`
- config updates no longer use `writeGlobalConfig()`

Use a mocked service factory instead of real `localStorage`.

**Step 2: Run test to verify it fails**

Run: `bun run test -- src/tradingPanelStore.test.tsx`

Expected: FAIL because the store still relies on `localStorage` and file config helpers.

**Step 3: Write minimal implementation**

In `src/tradingPanelStore.ts`:
- remove `readTradingPanelState()` and `writeTradingPanelSessionState()`
- remove `storage` event synchronization
- remove primary-path dependency on `globalConfigStorage.ts`
- inject or create the trading panel service
- hydrate state asynchronously on mount
- avoid overwriting persisted state before hydration completes
- preserve the public hook shape used by UI components

**Step 4: Run test to verify it passes**

Run: `bun run test -- src/tradingPanelStore.test.tsx`

Expected: PASS.

**Step 5: Commit**

```bash
git add src/tradingPanelStore.ts src/tradingPanelStore.test.tsx
git commit -m "refactor: load trading panel state through service"
```

### Task 7: Remove `localStorage`-based state seeding from app tests

**Files:**
- Modify: `src/tradingPanelApp.test.tsx`

**Step 1: Write the failing test**

Rewrite one test at a time so it no longer seeds `window.localStorage` and instead mocks the store or service state directly.

**Step 2: Run test to verify it fails**

Run: `bun run test -- src/tradingPanelApp.test.tsx`

Expected: FAIL while tests still depend on `seedState()` and persistence keys.

**Step 3: Write minimal implementation**

Replace `seedState()` usage with a mocked `useTradingPanelState()` or a mocked service-backed hook setup.

Delete imports of:
- `TRADING_PANEL_STORAGE_KEY`
- `GLOBAL_CONFIG_FALLBACK_STORAGE_KEY`

Keep the UI assertions unchanged where possible.

**Step 4: Run test to verify it passes**

Run: `bun run test -- src/tradingPanelApp.test.tsx`

Expected: PASS.

**Step 5: Commit**

```bash
git add src/tradingPanelApp.test.tsx
git commit -m "test: remove localstorage seeding from app tests"
```

### Task 8: Remove obsolete storage keys from model exports

**Files:**
- Modify: `src/model/storage.ts`
- Modify: `src/model/storage.test.ts`

**Step 1: Write the failing test**

Update `src/model/storage.test.ts` so it no longer asserts `TRADING_PANEL_STORAGE_KEY` and instead checks only the remaining persisted model exports such as `PANEL_BOUNDS_STORAGE_KEY`, `TradingPanelState`, and chart state typing.

**Step 2: Run test to verify it fails**

Run: `bun run test -- src/model/storage.test.ts`

Expected: FAIL because the test still references the obsolete key.

**Step 3: Write minimal implementation**

In `src/model/storage.ts`:
- remove `TRADING_PANEL_STORAGE_KEY`
- remove any `StorageSchema` entries tied to localStorage state persistence
- keep `PANEL_BOUNDS_STORAGE_KEY` because panel bounds are still persisted separately

Update imports elsewhere as needed.

**Step 4: Run test to verify it passes**

Run: `bun run test -- src/model/storage.test.ts`

Expected: PASS.

**Step 5: Commit**

```bash
git add src/model/storage.ts src/model/storage.test.ts
git commit -m "refactor: remove localstorage state key from storage models"
```

### Task 9: Full verification

**Files:**
- No intended code changes

**Step 1: Run targeted tests**

Run: `bun run test -- src/tradingPanelRepository.test.ts src/tradingPanelService.test.ts src/tradingPanelStore.test.tsx src/tradingPanelDomain.test.ts`

Expected: PASS.

**Step 2: Run app-level regression tests**

Run: `bun run test -- src/tradingPanelApp.test.tsx src/main.test.tsx`

Expected: PASS.

**Step 3: Run full test suite**

Run: `bun run test`

Expected: PASS with no unexpected warnings.

**Step 4: Run build**

Run: `bun run build`

Expected: TypeScript and Vite build both succeed.

**Step 5: Commit**

```bash
git add .
git commit -m "feat: persist trading panel state with sqlite service layer"
```

### Notes

- Do not reintroduce `localStorage` fallback for trading panel state.
- Do not move business rules out of `src/tradingPanelDomain.ts`.
- Keep SQL column naming in `snake_case` and TypeScript models in `camelCase`.
- Use `zod` parsing at service inputs and at SQL row/config boundaries.
- If plugin-sql API details are unclear during implementation, check current docs before coding.
