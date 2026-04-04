# Tauri Detail Topmost Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a toggle in each detail window that lets the user turn always-on-top on and off.

**Architecture:** Keep the behavior in the frontend detail view and use the current Tauri window API directly. Wrap the current-window calls in a tiny helper so the UI stays small and the core logic is unit-testable.

**Tech Stack:** Tauri v2, React 19, TypeScript, Vitest

---

### Task 1: Add failing tests for topmost window control

**Files:**
- Create: `src/detailTopmost.test.ts`
- Create: `src/detailTopmost.ts`

**Step 1: Write the failing test**

Cover these behaviors:

- reads the current always-on-top state
- toggles from off to on
- toggles from on to off

**Step 2: Run test to verify it fails**

Run: `bunx vitest run src/detailTopmost.test.ts`
Expected: FAIL because the helper module does not exist yet.

**Step 3: Write minimal implementation**

Implement a helper that wraps the current window methods behind a small interface.

**Step 4: Run test to verify it passes**

Run: `bunx vitest run src/detailTopmost.test.ts`
Expected: PASS

### Task 2: Connect the toggle to the detail view

**Files:**
- Modify: `src/App.tsx`
- Modify: `src/App.css`

**Step 1: Keep the UI change minimal**

Add a button or switch-like control to the detail page only.

**Step 2: Read initial state on mount**

Load the current always-on-top value and reflect it in the control.

**Step 3: Toggle on user action**

Call the helper to update the current window and refresh local state.

**Step 4: Show a small inline error on failure**

Do not add toast infrastructure.

### Task 3: Update Tauri capability permissions

**Files:**
- Modify: `src-tauri/capabilities/default.json`

**Step 1: Add permissions needed for always-on-top**

Include the read and write permissions for always-on-top state.

**Step 2: Verify the config still loads**

Run: `bun run tauri --help`
Expected: PASS

### Task 4: Final verification

**Files:**
- Modify: none

**Step 1: Run focused tests**

Run: `bunx vitest run src/detailWindow.test.ts src/detailTopmost.test.ts`
Expected: PASS

**Step 2: Run frontend build**

Run: `bun run build`
Expected: PASS

**Step 3: Manual runtime check**

Run: `bun run tauri dev`
Expected:

- detail page shows a topmost toggle
- turning it on keeps the window above others
- turning it off restores normal stacking
