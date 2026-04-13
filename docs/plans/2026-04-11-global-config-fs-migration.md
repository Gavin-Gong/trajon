# Global Config FS Migration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove the temporary fs demo UI and migrate global configuration persistence from `localStorage` to `~/.trading-flow/config.json` while keeping session state in `localStorage`.

**Architecture:** Add a dedicated config storage helper for `config.json`, then split `useTradingPanelState` persistence responsibilities into file-backed config fields and localStorage-backed session fields. Preserve existing user config by reading legacy localStorage config when the file is absent, and remove the analysis-page demo card and its tests.

**Tech Stack:** React 19, TypeScript, Vitest, Testing Library, Tauri fs plugin

---

### Task 1: Add failing tests for file-backed global config storage

**Files:**
- Create: `src/globalConfigStorage.test.ts`
- Create: `src/globalConfigStorage.ts`
- Read for context: `src/model/globalConfig.ts`
- Read for context: `src/tradingPanelStore.ts`

**Step 1: Write the failing test**

Add tests for a new helper that manages `~/.trading-flow/config.json` and covers:

- reading stored config from the fs helper,
- returning `null` when `config.json` is missing,
- writing normalized config to `config.json`,
- browser/test fallback behavior when Tauri fs is unavailable.

Keep the config shape limited to:

- `config`
- `pairPresets`
- `defaultPresetSymbol`

**Step 2: Run test to verify it fails**

Run: `bunx vitest run src/globalConfigStorage.test.ts --exclude ".worktrees/**"`
Expected: FAIL because `src/globalConfigStorage.ts` does not exist yet.

**Step 3: Write minimal implementation**

Create `src/globalConfigStorage.ts` with:

- constants for `.trading-flow/config.json`,
- read helper,
- write helper,
- small normalization helpers for the config subset,
- browser/test fallback storage path that does not require Tauri runtime.

**Step 4: Run test to verify it passes**

Run: `bunx vitest run src/globalConfigStorage.test.ts --exclude ".worktrees/**"`
Expected: PASS.

**Step 5: Commit**

```bash
git add src/globalConfigStorage.ts src/globalConfigStorage.test.ts
git commit -m "feat: add file-backed global config storage"
```

### Task 2: Add failing store tests for config/file split persistence

**Files:**
- Modify: `src/tradingPanelStore.test.tsx`
- Modify: `src/tradingPanelStore.ts`
- Supporting context: `src/globalConfigStorage.ts`

**Step 1: Write the failing test**

Add tests covering:

- config actions write through the config storage helper,
- non-config actions still persist only session state to `localStorage`,
- legacy localStorage config is used when file config is absent,
- file config overrides legacy localStorage config when present.

Mock the new config storage helper in the test file instead of touching the real fs layer.

**Step 2: Run test to verify it fails**

Run: `bunx vitest run src/tradingPanelStore.test.tsx --exclude ".worktrees/**"`
Expected: FAIL because the store still writes the full state to `localStorage` and does not read file-backed config.

**Step 3: Write minimal implementation**

Update `src/tradingPanelStore.ts` to:

- split persistence into session-only localStorage state and file-backed config state,
- initialize from localStorage plus file config,
- fall back to legacy localStorage config when file config is missing,
- keep config writes going through the new helper,
- keep `journals`, `chart`, and `panelPinned` in localStorage.

Keep runtime behavior safe in browser/tests.

**Step 4: Run test to verify it passes**

Run: `bunx vitest run src/tradingPanelStore.test.tsx --exclude ".worktrees/**"`
Expected: PASS.

**Step 5: Commit**

```bash
git add src/tradingPanelStore.ts src/tradingPanelStore.test.tsx src/globalConfigStorage.ts
git commit -m "feat: migrate global config persistence to fs"
```

### Task 3: Add failing UI tests for demo removal

**Files:**
- Modify: `src/tradingPanelApp.test.tsx`
- Modify: `src/components/tradingPanel/TradingPanelMain.tsx`

**Step 1: Write the failing test**

Replace the analysis-page fs demo assertions with tests that confirm the analysis page no longer renders:

- `Tauri FS Demo`
- `写入示例`
- `读取示例`

Keep the rest of the analysis-page tests intact.

**Step 2: Run test to verify it fails**

Run: `bunx vitest run src/tradingPanelApp.test.tsx --exclude ".worktrees/**"`
Expected: FAIL because the demo card still exists.

**Step 3: Write minimal implementation**

Remove the demo card and its related local state/handlers/imports from `AnalyticsPanel` in `src/components/tradingPanel/TradingPanelMain.tsx`.

**Step 4: Run test to verify it passes**

Run: `bunx vitest run src/tradingPanelApp.test.tsx --exclude ".worktrees/**"`
Expected: PASS.

**Step 5: Commit**

```bash
git add src/components/tradingPanel/TradingPanelMain.tsx src/tradingPanelApp.test.tsx
git commit -m "refactor: remove temporary fs demo card"
```

### Task 4: Verify integrated behavior

**Files:**
- No additional code changes unless verification reveals issues.

**Step 1: Run focused tests**

Run: `bunx vitest run src/globalConfigStorage.test.ts src/tradingPanelStore.test.tsx src/tradingPanelApp.test.tsx --exclude ".worktrees/**"`
Expected: PASS.

**Step 2: Run frontend build**

Run: `npm run build`
Expected: PASS.

**Step 3: Run Rust verification**

Run: `cargo check`
Expected: PASS.

**Step 4: Record current full-suite status**

Run: `bun run test`
Expected: likely still FAIL because `.worktrees/**` and unrelated pre-existing tests are still discovered. Report exact failures if unchanged; do not expand scope unless asked.

**Step 5: Commit**

```bash
git add src/globalConfigStorage.ts src/globalConfigStorage.test.ts src/tradingPanelStore.ts src/tradingPanelStore.test.tsx src/components/tradingPanel/TradingPanelMain.tsx src/tradingPanelApp.test.tsx
git commit -m "chore: verify fs-backed global config migration"
```
