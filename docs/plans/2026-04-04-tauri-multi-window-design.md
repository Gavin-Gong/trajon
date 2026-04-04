# Tauri Multi-Window Detail Design

## Goal

Allow the main Tauri window to open multiple independent detail windows while enforcing a single open window per `symbol`.

## Decisions

- Use Tauri v2 `WebviewWindow` from the frontend.
- Keep the current single-page starter structure and avoid adding a router.
- Encode detail state in the URL query string as `/?view=detail&symbol=BTCUSDT`.
- Derive a deterministic window label from `symbol` so each symbol maps to exactly one detail window.
- When a detail window for the same symbol already exists, reuse it by showing and focusing it.

## Why This Approach

This is the smallest change that satisfies the requested behavior. It works with the current starter app, keeps refresh-safe state in the URL, and avoids introducing Rust-side window orchestration before it is needed.

## Frontend Structure

- `App.tsx` will choose between rendering the main launcher view and the detail view based on `window.location.search`.
- A small window helper module will own symbol normalization, label generation, detail URL generation, and the open-or-focus behavior.
- The main view will collect a symbol and call the helper.
- The detail view will read the symbol from the URL and render a simple independent page.

## Window Rules

- Different symbols can open at the same time.
- The same symbol can only have one window.
- Symbols are normalized before use so `btcusdt` and ` BTCUSDT ` target the same window.

## Permissions

The Tauri capability config will allow both the main window and `detail-*` windows to access the default core permissions.

## Testing Strategy

- Add unit tests for symbol normalization, label generation, detail URL generation, and search parsing.
- Add unit tests for the window opening logic with a small fake window API to verify create-vs-focus behavior.

## Out Of Scope

- Window-to-window event communication.
- Shared application state across windows.
- Rich detail content or live trading data.
