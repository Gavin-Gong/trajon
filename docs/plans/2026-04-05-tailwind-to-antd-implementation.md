# Tailwind-to-Ant Design Migration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove Tailwind from the desktop app and migrate the current mixed UI to a more standard Ant Design structure without breaking existing journal workflows.

**Architecture:** Keep the migration centered in `src/tradingPanelApp.tsx`, add regression tests before UI rewrites, introduce root-level Ant Design theming with `ConfigProvider`, then replace Tailwind-heavy layout/composition with Ant Design primitives before removing Tailwind from the toolchain. Avoid broad domain refactors and keep behavior stable.

**Tech Stack:** React 19, TypeScript, Vite 7, Vitest, Testing Library, Ant Design 6, Tauri.

---

### Task 1: Expand regression coverage before refactor

**Files:**
- Modify: `src/tradingPanelApp.test.tsx`
- Modify: `src/test/setup.ts`
- Test: `src/tradingPanelApp.test.tsx`

**Step 1: Write the failing test**

Add one or more assertions that capture the current app shell and key flows that must survive the migration, for example:

```tsx
it("renders navigation and main workspace sections", () => {
  window.localStorage.setItem(TRADING_PANEL_STORAGE_KEY, JSON.stringify(buildState()));

  render(<TradingPanelDesktop view="main" />);

  expect(screen.getByRole("menu")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "打开记录面板" })).toBeInTheDocument();
  expect(screen.getByText("今日执行概览")).toBeInTheDocument();
  expect(screen.getByText("活跃标的列表")).toBeInTheDocument();
});
```

If Ant Design layout primitives require extra browser API mocks, add only the missing test setup support to `src/test/setup.ts`.

**Step 2: Run test to verify it fails**

Run: `npm test -- src/tradingPanelApp.test.tsx`

Expected: FAIL because at least one new assertion targets behavior not yet guaranteed or because setup support is incomplete.

**Step 3: Write minimal implementation**

Add only the missing test support or accessibility-preserving markup needed for the current app to satisfy the new regression checks.

**Step 4: Run test to verify it passes**

Run: `npm test -- src/tradingPanelApp.test.tsx`

Expected: PASS.

**Step 5: Commit**

Do not commit unless explicitly requested by the user.

---

### Task 2: Introduce Ant Design root theme foundation

**Files:**
- Modify: `src/App.tsx`
- Modify: `src/main.tsx`
- Modify: `src/App.css`
- Test: `src/tradingPanelApp.test.tsx`

**Step 1: Write the failing test**

Add a test that proves the app still renders correctly when wrapped with the new root provider assumptions, for example by checking the same desktop shell still renders after provider introduction.

```tsx
it("renders the desktop shell through the app entry provider", () => {
  window.localStorage.setItem(TRADING_PANEL_STORAGE_KEY, JSON.stringify(buildState()));

  render(<App />);

  expect(screen.getByText("盘中复盘工作台")).toBeInTheDocument();
});
```

**Step 2: Run test to verify it fails**

Run: `npm test -- src/tradingPanelApp.test.tsx`

Expected: FAIL before provider wiring is added.

**Step 3: Write minimal implementation**

- Wrap the app with `ConfigProvider`
- Set minimal theme tokens such as `colorPrimary`, `borderRadius`, and typography-friendly defaults
- Reduce `src/App.css` to base document styling plus any truly necessary app-scoped global rules

Example direction:

```tsx
<ConfigProvider
  theme={{
    token: {
      colorPrimary: "#1677ff",
      borderRadius: 12,
    },
  }}
>
  <TradingPanelDesktop view={view} />
</ConfigProvider>
```

**Step 4: Run test to verify it passes**

Run: `npm test -- src/tradingPanelApp.test.tsx`

Expected: PASS.

**Step 5: Commit**

Do not commit unless explicitly requested by the user.

---

### Task 3: Migrate main desktop shell and analytics section

**Files:**
- Modify: `src/tradingPanelApp.tsx`
- Test: `src/tradingPanelApp.test.tsx`

**Step 1: Write the failing test**

Add or extend tests to lock the analytics shell behavior and content hierarchy, for example:

```tsx
it("shows analytics summary content in the main desktop view", () => {
  window.localStorage.setItem(TRADING_PANEL_STORAGE_KEY, JSON.stringify(buildState()));

  render(<TradingPanelDesktop view="main" />);

  expect(screen.getByText("今日执行概览")).toBeInTheDocument();
  expect(screen.getByText("当前追踪标的")).toBeInTheDocument();
  expect(screen.getByText("活跃标的列表")).toBeInTheDocument();
});
```

**Step 2: Run test to verify it fails**

Run: `npm test -- src/tradingPanelApp.test.tsx`

Expected: FAIL after first structural rewrite attempt if the migrated shell does not preserve these sections correctly.

**Step 3: Write minimal implementation**

Replace Tailwind-heavy composition in the main view with Ant Design composition:

- use `Layout`, `Card`, `Flex`, `Space`, `Typography`, `Statistic`, `List`, `Divider`, and `Tag` where appropriate
- remove Tailwind class-based shells and spacing wrappers
- preserve menu-driven section switching and the floating panel entry action

**Step 4: Run test to verify it passes**

Run: `npm test -- src/tradingPanelApp.test.tsx`

Expected: PASS.

**Step 5: Commit**

Do not commit unless explicitly requested by the user.

---

### Task 4: Migrate calendar, settings, and panel surfaces

**Files:**
- Modify: `src/tradingPanelApp.tsx`
- Test: `src/tradingPanelApp.test.tsx`

**Step 1: Write the failing test**

Add focused tests for the non-analytics surfaces that should remain available after migration, such as settings labels, panel controls, or calendar/history headings.

```tsx
it("renders panel mode quick capture actions", () => {
  window.localStorage.setItem(TRADING_PANEL_STORAGE_KEY, JSON.stringify(buildState()));

  render(<TradingPanelDesktop view="panel" />);

  expect(screen.getByRole("button", { name: /开始 journal/i })).toBeInTheDocument();
});
```

Adjust the expected accessible names to match the actual intended UI labels.

**Step 2: Run test to verify it fails**

Run: `npm test -- src/tradingPanelApp.test.tsx`

Expected: FAIL before the migrated panel/calendar/settings surfaces preserve the required controls.

**Step 3: Write minimal implementation**

- convert remaining Tailwind-wrapped areas toward Ant Design composition
- prefer Ant `Form`, `Input`, `InputNumber`, `Segmented`, `Card`, `List`, `Descriptions`, `Space`, `Flex`, `Empty`, and `Typography`
- keep the current domain behavior and event creation logic intact

**Step 4: Run test to verify it passes**

Run: `npm test -- src/tradingPanelApp.test.tsx`

Expected: PASS.

**Step 5: Commit**

Do not commit unless explicitly requested by the user.

---

### Task 5: Remove Tailwind integration and clean global styling

**Files:**
- Modify: `src/App.css`
- Modify: `vite.config.ts`
- Modify: `package.json`
- Modify: `package-lock.json`
- Test: `src/tradingPanelApp.test.tsx`

**Step 1: Write the failing test**

Add a narrow regression assertion only if needed to ensure the app still renders after global CSS/tooling cleanup. If existing tests already cover the rendered shell, reuse them rather than adding redundant coverage.

**Step 2: Run test to verify it fails**

Run: `npm test -- src/tradingPanelApp.test.tsx`

Expected: FAIL if cleanup accidentally removes required runtime styling/import support.

**Step 3: Write minimal implementation**

- remove Tailwind import from `src/App.css`
- remove `@tailwindcss/vite` usage from `vite.config.ts`
- remove Tailwind dependencies from `package.json`
- update lockfile via install so dependency state matches `package.json`
- keep only the global CSS that is still necessary after Ant Design migration

**Step 4: Run test to verify it passes**

Run: `npm test -- src/tradingPanelApp.test.tsx`

Expected: PASS.

**Step 5: Commit**

Do not commit unless explicitly requested by the user.

---

### Task 6: Verify diagnostics, tests, and production build

**Files:**
- Modify only if verification exposes issues in:
  - `src/App.tsx`
  - `src/main.tsx`
  - `src/App.css`
  - `src/tradingPanelApp.tsx`
  - `src/tradingPanelApp.test.tsx`
  - `vite.config.ts`
  - `package.json`

**Step 1: Run diagnostics on changed files**

Run diagnostics for all edited source files and resolve any errors before moving on.

**Step 2: Run targeted tests**

Run: `npm test -- src/tradingPanelApp.test.tsx src/panelWindow.test.ts`

Expected: PASS.

**Step 3: Run full test suite**

Run: `npm test`

Expected: PASS.

**Step 4: Run production build**

Run: `npm run build`

Expected: PASS.

**Step 5: Commit**

Do not commit unless explicitly requested by the user.
