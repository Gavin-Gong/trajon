## Background

The project currently persists trading panel state in `localStorage` via `src/tradingPanelStore.ts`. There is no existing Tauri file-system abstraction in the frontend. The Tauri `fs` plugin has been installed and scoped to `$HOME/.trading-flow` with read and write access.

The requested next step is a standalone example that demonstrates reading and writing a file under `~/.trading-flow` without changing current product persistence behavior.

## Goals

- Add a minimal frontend helper for file IO through `@tauri-apps/plugin-fs`.
- Keep all current trading panel behavior unchanged.
- Target a fixed demo file path under `~/.trading-flow/demo.json`.
- Remain safe in browser and test environments where Tauri APIs are unavailable.

## Non-Goals

- Replacing `localStorage` persistence.
- Introducing a full repository-wide storage abstraction.
- Adding new production UI flows unless needed for a tiny demo trigger.

## Approach Options

### Option 1: Standalone helper module

Create a dedicated module such as `src/fileStorage.ts` that exports a small set of functions:

- `ensureDemoDir()`
- `readDemoJson()`
- `writeDemoJson(data)`

Pros:

- Minimal surface area.
- Easy to test.
- Easy to expand later into real persistence.

Cons:

- Not visible in the UI by itself.

### Option 2: Single utility function only

Place all logic into one exported helper and keep the example purely code-level.

Pros:

- Smallest code change.

Cons:

- Less clear API shape.
- Harder to extend cleanly.

### Option 3: UI demo wired into existing app

Add a button or panel action that reads and writes the demo file.

Pros:

- Easiest manual verification.

Cons:

- Expands scope into product UI.
- Adds behavior unrelated to current feature request.

## Selected Design

Use Option 1.

Add a standalone helper module that owns the demo directory and file path. The module will use Tauri `fs` APIs only when running in a Tauri runtime. In browser or test contexts it will fail predictably with a clear error so existing tests and browser fallback behavior remain intact.

The helper will:

- ensure `~/.trading-flow` exists before write attempts,
- store JSON in `demo.json`,
- read and parse JSON from that file,
- return explicit errors for missing Tauri runtime or invalid JSON.

## API Shape

Planned exports:

- `DEMO_DIRECTORY = ".trading-flow"`
- `DEMO_FILE_NAME = "demo.json"`
- `isTauriFsAvailable()`
- `readDemoJson()`
- `writeDemoJson(value)`

The implementation should prefer a narrow API over a generic storage layer.

## Environment Behavior

- Tauri runtime: real file operations through `@tauri-apps/plugin-fs` with `BaseDirectory.Home`.
- Browser runtime: reject with a stable error explaining that file storage requires Tauri.
- Test runtime: same as browser unless explicitly mocked in tests.

## Error Handling

- Missing Tauri runtime: throw an error with a clear message.
- Missing file: either return `null` or a dedicated result indicating absence.
- Invalid JSON content: throw an error naming the demo file.

For the first implementation, returning `null` for a missing file is the simplest behavior.

## Testing

Add tests for:

- browser/test fallback when Tauri runtime is unavailable,
- successful write path with mocked plugin functions,
- successful read path with mocked plugin functions,
- invalid JSON handling,
- missing file returning `null`.

## Verification

- `bun run test`
- `npm run build`
- `cargo check`
