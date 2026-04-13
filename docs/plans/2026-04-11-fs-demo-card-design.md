## Background

The project now includes a standalone file helper in `src/fileStorage.ts` for reading and writing `~/.trading-flow/demo.json` through the Tauri `fs` plugin. There is not yet any UI that exercises this helper.

The requested next step is to add a temporary demo entry in the desktop UI so the file helper can be triggered manually without changing the app's existing persistence model.

## Goals

- Add a minimal demo card in the main desktop UI.
- Keep the current navigation structure intact.
- Avoid changing `tradingPanelStore` or replacing `localStorage` persistence.
- Make it easy to manually verify read/write behavior in Tauri.

## Non-Goals

- Adding a permanent product feature.
- Introducing a new top-level section for file storage.
- Replacing current app persistence.

## Approach Options

### Option 1: Analysis page demo card

Add a small demo card below the existing analysis placeholder in `AnalyticsPanel`.

Pros:

- Minimal navigation impact.
- Easy to find.
- Clearly temporary and isolated.

Cons:

- Slightly mixes demo tooling into a placeholder page.

### Option 2: Settings page tool block

Add the demo under settings as a developer utility.

Pros:

- Semantically close to tools/config.

Cons:

- More clicks for manual verification.
- Slightly heavier scope because settings already carries actual product configuration.

### Option 3: Dedicated menu item

Add a new navigation entry such as `FS Demo`.

Pros:

- Most explicit.

Cons:

- Overkill for a temporary demo.
- Adds product-level navigation noise.

## Selected Design

Use Option 1.

Add a small card to `AnalyticsPanel` under the current empty placeholder. The card will contain:

- a short explanation that this is a temporary Tauri FS demo,
- a `写入示例` button,
- a `读取示例` button,
- a result area showing either the last successful JSON payload or the last error message.

## Behavior

### Write action

When the user clicks `写入示例`, write a fixed JSON-like object to `demo.json`, for example:

- `message: "Trading Flow demo"`
- `source: "trading-flow-demo"`
- `timestamp: <ISO string>`

After success, show the formatted payload in the result area.

### Read action

When the user clicks `读取示例`, call `readDemoJson()` and show:

- formatted JSON if present,
- an explicit `未找到 demo 文件` style message if the helper returns `null`.

### Error handling

If the helper throws, show the error message in the result area. This preserves useful browser-mode behavior where the helper reports that Tauri file storage is only available in the desktop app.

## UI Shape

- Reuse existing Ant Design `Card`, `Space`, `Flex`, `Button`, `Typography`.
- Keep styling consistent with the rest of `AnalyticsPanel` by using the existing shared card tokens.
- Do not add a new route or a new app section.

## State Management

Use local component state inside `AnalyticsPanel` for:

- current operation status,
- latest result text,
- latest error text.

This state is demo-only and should not enter `tradingPanelStore`.

## Testing

Add UI tests that verify:

- the demo card renders on the analysis page,
- clicking `写入示例` calls the helper and shows success output,
- clicking `读取示例` calls the helper and shows success output,
- missing file output is rendered when `readDemoJson()` returns `null`,
- thrown helper errors are rendered.

Mock `src/fileStorage.ts` in the UI test file instead of touching the real Tauri plugin.

## Verification

- `bun run test -- src/tradingPanelApp.test.tsx src/fileStorage.test.ts`
- `npm run build`
- `cargo check`
