# FS Standalone Demo Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a standalone frontend helper that reads and writes `~/.trading-flow/demo.json` through the Tauri `fs` plugin without changing existing app persistence.

**Architecture:** Introduce a narrow file-storage helper in `src/fileStorage.ts` and keep all current `localStorage`-backed trading state untouched. The helper will guard against non-Tauri environments, ensure the target directory exists, and expose dedicated read/write functions for a fixed demo JSON file.

**Tech Stack:** React 19, TypeScript, Vitest, Tauri v2, `@tauri-apps/plugin-fs`

---

### Task 1: Add failing tests for runtime guards and happy-path IO

**Files:**
- Create: `src/fileStorage.test.ts`
- Create: `src/fileStorage.ts`

**Step 1: Write the failing test**

Add tests for these behaviors:

```ts
it("throws when Tauri fs is unavailable", async () => {
  await expect(readDemoJson()).rejects.toThrow("Tauri file storage is only available in the desktop app");
});

it("writes demo json to the home-scoped file", async () => {
  await writeDemoJson({ hello: "world" });
  expect(mkdir).toHaveBeenCalled();
  expect(writeTextFile).toHaveBeenCalled();
});

it("reads parsed demo json from the home-scoped file", async () => {
  vi.mocked(readTextFile).mockResolvedValue('{"hello":"world"}');
  await expect(readDemoJson()).resolves.toEqual({ hello: "world" });
});
```

**Step 2: Run test to verify it fails**

Run: `bun run test -- src/fileStorage.test.ts`
Expected: FAIL because `src/fileStorage.ts` does not exist yet.

**Step 3: Write minimal implementation**

Create `src/fileStorage.ts` with:

- constants for directory and file name,
- a runtime guard,
- helpers to ensure the directory exists,
- `readDemoJson()` and `writeDemoJson(value)`.

Use `BaseDirectory.Home` and a relative path of `.trading-flow/demo.json`.

**Step 4: Run test to verify it passes**

Run: `bun run test -- src/fileStorage.test.ts`
Expected: PASS.

**Step 5: Commit**

```bash
git add src/fileStorage.ts src/fileStorage.test.ts
git commit -m "feat: add standalone tauri fs demo helper"
```

### Task 2: Cover missing-file and invalid-JSON behavior

**Files:**
- Modify: `src/fileStorage.test.ts`
- Modify: `src/fileStorage.ts`

**Step 1: Write the failing test**

Add tests for:

```ts
it("returns null when the demo file does not exist", async () => {
  vi.mocked(exists).mockResolvedValue(false);
  await expect(readDemoJson()).resolves.toBeNull();
});

it("throws a clear error when demo json is invalid", async () => {
  vi.mocked(exists).mockResolvedValue(true);
  vi.mocked(readTextFile).mockResolvedValue("not-json");
  await expect(readDemoJson()).rejects.toThrow("Invalid JSON in demo.json");
});
```

**Step 2: Run test to verify it fails**

Run: `bun run test -- src/fileStorage.test.ts`
Expected: FAIL because the initial implementation does not yet distinguish these cases.

**Step 3: Write minimal implementation**

Update `readDemoJson()` to:

- return `null` when the file is absent,
- parse JSON when present,
- throw a specific invalid-JSON error for parse failures.

**Step 4: Run test to verify it passes**

Run: `bun run test -- src/fileStorage.test.ts`
Expected: PASS.

**Step 5: Commit**

```bash
git add src/fileStorage.ts src/fileStorage.test.ts
git commit -m "test: cover demo fs read edge cases"
```

### Task 3: Verify repo-wide safety

**Files:**
- No code changes required unless verification reveals issues.

**Step 1: Run focused tests**

Run: `bun run test -- src/fileStorage.test.ts`
Expected: PASS.

**Step 2: Run full test suite**

Run: `bun run test`
Expected: PASS.

**Step 3: Run frontend build**

Run: `npm run build`
Expected: PASS.

**Step 4: Run Rust verification**

Run: `cargo check`
Expected: PASS.

**Step 5: Commit**

```bash
git add src/fileStorage.ts src/fileStorage.test.ts
git commit -m "chore: verify standalone fs demo helper"
```
