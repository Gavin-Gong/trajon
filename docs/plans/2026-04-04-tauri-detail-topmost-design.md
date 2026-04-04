# Tauri Detail Topmost Design

## Goal

Allow the user to toggle the current detail window's always-on-top state from within the detail page.

## Decisions

- Add the control only to detail windows.
- Use a local toggle control bound to the current Tauri window.
- Read the initial window topmost state when the detail page loads.
- Update the window by calling the current window API directly from the frontend.
- Show a small inline error if the toggle operation fails.

## Why This Approach

The behavior belongs to the detail window itself, so the smallest correct implementation is to let the detail page manage the current window state directly. This avoids unnecessary Rust-side coordination while leaving room for future global window management.

## Frontend Structure

- Add a small helper module for detail window topmost state.
- The helper will wrap `getCurrentWebviewWindow`, `isAlwaysOnTop`, and `setAlwaysOnTop` behind a testable interface.
- `App.tsx` detail mode will render a toggle button and sync its state on load.

## Permissions

The existing capability config needs the window permissions required to read and change always-on-top state.

## Testing Strategy

- Add focused unit tests for reading and toggling the topmost state through a fake window API.
- Re-run the existing detail window tests to make sure the multi-window behavior still works.

## Out Of Scope

- Persisting topmost preference across app restarts.
- Global topmost controls from the main window.
- Cross-window synchronization of topmost state.
