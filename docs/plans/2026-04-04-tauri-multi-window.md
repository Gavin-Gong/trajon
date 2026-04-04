# Tauri Multi-Window Detail Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Let the main window open multiple detail windows while ensuring each `symbol` maps to one open window.

**Architecture:** Keep the React starter as a single entry point and switch rendered content based on URL query parameters. Put symbol normalization and window lifecycle logic in a small helper module so the core behavior is unit-testable without Tauri runtime integration.

**Tech Stack:** Tauri v2, React 19, TypeScript, Vitest

---

### Task 1: Add test support and document the feature

**Files:**
- Create: `docs/plans/2026-04-04-tauri-multi-window-design.md`
- Create: `docs/plans/2026-04-04-tauri-multi-window.md`
- Modify: `package.json`

**Step 1: Add the test runner dependency**

Install `vitest` as a dev dependency.

**Step 2: Verify the test command can run**

Run: `bunx vitest --run`
Expected: exits successfully or reports no tests found.

### Task 2: Write failing tests for detail window helpers

**Files:**
- Create: `src/detailWindow.test.ts`
- Create: `src/detailWindow.ts`

**Step 1: Write failing tests**

Cover these behaviors:

- symbols are trimmed and uppercased
- labels are deterministic and safe for Tauri window labels
- detail URLs include `view=detail` and `symbol`
- search parsing distinguishes main and detail views
- existing windows are shown and focused instead of recreated
- new symbols create new windows

**Step 2: Run test to verify it fails**

Run: `bunx vitest run src/detailWindow.test.ts`
Expected: FAIL because the helper module does not exist yet.

**Step 3: Write minimal implementation**

Implement the helper module with pure functions and one `openDetailWindow` function that depends on a tiny window API interface.

**Step 4: Run test to verify it passes**

Run: `bunx vitest run src/detailWindow.test.ts`
Expected: PASS

### Task 3: Connect the helper to the UI

**Files:**
- Modify: `src/App.tsx`
- Modify: `src/App.css`

**Step 1: Write the failing UI behavior test if practical**

If a DOM test is not worth the setup cost, rely on the helper tests and keep the UI change minimal.

**Step 2: Implement the main view**

Replace the starter greet demo with a simple symbol input and open button.

**Step 3: Implement the detail view**

Render a detail panel when the URL indicates `view=detail`.

**Step 4: Verify app build**

Run: `bun run build`
Expected: PASS

### Task 4: Update Tauri capability config

**Files:**
- Modify: `src-tauri/capabilities/default.json`

**Step 1: Add detail window coverage**

Allow `detail-*` windows to use the same default capability as `main`.

**Step 2: Verify Tauri build configuration still loads**

Run: `bun run tauri --help`
Expected: PASS

### Task 5: Final verification

**Files:**
- Modify: `README.md` if needed

**Step 1: Run focused tests**

Run: `bunx vitest run src/detailWindow.test.ts`
Expected: PASS

**Step 2: Run full frontend build**

Run: `bun run build`
Expected: PASS

**Step 3: Manual runtime check**

Run: `bun run tauri dev`
Expected:

- opening `BTCUSDT` creates one window
- opening `BTCUSDT` again focuses the same window
- opening `ETHUSDT` creates a second window
- refreshing a detail window keeps the symbol visible
