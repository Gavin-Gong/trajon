# FS Demo Card Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a temporary demo card on the analysis page that reads and writes `~/.trading-flow/demo.json` through the standalone Tauri fs helper.

**Architecture:** Keep the demo isolated inside `AnalyticsPanel` in `TradingPanelMain.tsx` and use local component state for transient UI feedback. Reuse the existing `src/fileStorage.ts` helper and mock it from `src/tradingPanelApp.test.tsx` so the UI tests never touch the real Tauri runtime.

**Tech Stack:** React 19, TypeScript, Ant Design, Vitest, Testing Library, Tauri fs helper

---

### Task 1: Add failing UI tests for the demo card

**Files:**
- Modify: `src/tradingPanelApp.test.tsx`
- Read for context: `src/components/tradingPanel/TradingPanelMain.tsx:150-210`
- Read for context: `src/fileStorage.ts`

**Step 1: Write the failing test**

Add tests to `src/tradingPanelApp.test.tsx` that:

- confirm a temporary fs demo card is rendered on the analysis page,
- click `写入示例` and verify the helper is called and success output is shown,
- click `读取示例` and verify success output is shown,
- verify `null` from `readDemoJson()` renders a missing-file message,
- verify thrown helper errors are rendered.

Mock `./fileStorage` at the top of the test file.

Example expectations:

```ts
expect(screen.getByText("Tauri FS Demo")).toBeInTheDocument();
fireEvent.click(screen.getByRole("button", { name: "写入示例" }));
await waitFor(() => {
  expect(writeDemoJson).toHaveBeenCalledTimes(1);
});
```

**Step 2: Run test to verify it fails**

Run: `bun run test -- src/tradingPanelApp.test.tsx`
Expected: FAIL because the demo card and its handlers do not exist yet.

**Step 3: Write minimal implementation**

Do not implement yet. This task stops after red.

**Step 4: Run test to verify it still fails for the intended reason**

Run: `bun run test -- src/tradingPanelApp.test.tsx`
Expected: FAIL on missing demo UI.

**Step 5: Commit**

```bash
git add src/tradingPanelApp.test.tsx
git commit -m "test: add fs demo card coverage"
```

### Task 2: Implement the analysis-page demo card

**Files:**
- Modify: `src/components/tradingPanel/TradingPanelMain.tsx`
- Reuse: `src/fileStorage.ts`
- Test: `src/tradingPanelApp.test.tsx`

**Step 1: Write the failing test**

Use the tests from Task 1 as the active failing spec.

**Step 2: Run test to verify it fails**

Run: `bun run test -- src/tradingPanelApp.test.tsx`
Expected: FAIL.

**Step 3: Write minimal implementation**

Update `AnalyticsPanel` in `src/components/tradingPanel/TradingPanelMain.tsx` to add a new card below the existing placeholder card and preset card.

Implementation requirements:

- import `readDemoJson` and `writeDemoJson`,
- add local UI state for the latest demo output,
- add `写入示例` button that writes a payload with `message`, `source`, and `timestamp`,
- add `读取示例` button that loads the helper result,
- render formatted JSON on success,
- render a missing-file message when `readDemoJson()` returns `null`,
- render thrown error messages.

Keep styles aligned with existing Ant Design cards and text blocks.

**Step 4: Run test to verify it passes**

Run: `bun run test -- src/tradingPanelApp.test.tsx`
Expected: PASS for the new demo-card tests.

**Step 5: Commit**

```bash
git add src/components/tradingPanel/TradingPanelMain.tsx src/tradingPanelApp.test.tsx
git commit -m "feat: add analysis page fs demo card"
```

### Task 3: Verify helper and UI together

**Files:**
- No additional code changes unless verification reveals issues.

**Step 1: Run focused helper and UI tests**

Run: `bun run test -- src/fileStorage.test.ts src/tradingPanelApp.test.tsx`
Expected: PASS.

**Step 2: Run frontend build**

Run: `npm run build`
Expected: PASS.

**Step 3: Run Rust verification**

Run: `cargo check`
Expected: PASS.

**Step 4: Note current full-suite status**

Run: `bun run test`
Expected: likely still FAIL due to existing unrelated `.worktrees/**` test discovery and other pre-existing failures. Record but do not expand scope unless asked.

**Step 5: Commit**

```bash
git add src/components/tradingPanel/TradingPanelMain.tsx src/tradingPanelApp.test.tsx src/fileStorage.ts src/fileStorage.test.ts
git commit -m "chore: verify fs demo card integration"
```
