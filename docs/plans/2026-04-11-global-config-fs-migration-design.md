## Background

The app currently persists the entire `TradingPanelState` into `localStorage` from `src/tradingPanelStore.ts`. Recent work also added a standalone Tauri fs helper and a temporary analysis-page demo card.

The next requested change is to remove the demo and migrate only global configuration data to file storage, while leaving session-oriented state in `localStorage`.

## Goals

- Remove the temporary fs demo UI.
- Move configuration-related persistence from `localStorage` to `~/.trading-flow/config.json`.
- Keep journals, chart state, and panel pin state in `localStorage`.
- Preserve existing user configuration during migration.

## Non-Goals

- Moving the full trading panel state to files.
- Changing window-bounds persistence.
- Introducing a backend or SQL persistence path for config.

## Selected Scope

Persist these fields in `~/.trading-flow/config.json`:

- `config`
- `pairPresets`
- `defaultPresetSymbol`

Keep these fields in `localStorage`:

- `journals`
- `chart`
- `panelPinned`

## File Location

- Directory: `~/.trading-flow`
- File: `config.json`

The file will use JSON matching the normalized subset of state needed for configuration.

## Architecture

### 1. Dedicated config storage module

Add a dedicated frontend storage helper, e.g. `src/globalConfigStorage.ts`, responsible for:

- reading config from `config.json`,
- writing config to `config.json`,
- exposing a browser/test fallback path when Tauri fs is unavailable,
- normalizing incoming config payloads.

### 2. Split persisted state model

`src/tradingPanelStore.ts` will stop treating `localStorage` as the source of truth for config fields.

On initialization:

1. Read legacy/full state from `localStorage` as today.
2. Read config from `config.json`.
3. Build the final in-memory state as:
   - non-config fields from `localStorage`,
   - config fields from file if available,
   - otherwise config fields from the legacy local snapshot.

On updates:

- config-related actions write config to file,
- non-config actions keep writing the session subset to `localStorage`.

### 3. Migration behavior

If `config.json` does not exist yet, the app should still start with the config values found in legacy `localStorage`.

The first config write will persist those values into `config.json`.

This avoids destructive one-time migration logic and minimizes risk. Old localStorage config data may remain present, but runtime precedence becomes:

1. file config
2. legacy local config
3. defaults

## Runtime Behavior

### Tauri desktop

- Use real fs reads/writes to `~/.trading-flow/config.json`.

### Browser and tests

- Do not hard-require Tauri runtime.
- Allow a local fallback for tests and browser execution so current test patterns remain viable.
- The fallback should be isolated to the config storage helper, not spread through UI code.

## UI Impact

- Remove the `Tauri FS Demo` card from `AnalyticsPanel`.
- Remove the corresponding demo interaction tests.
- No new UI is required for the config migration itself.

## Error Handling

- Invalid or unreadable file config should fall back to normalized defaults or legacy local config.
- Write failures should not crash rendering.
- Prefer preserving last known in-memory state even if persistence fails.

## Testing

### New tests

- `src/globalConfigStorage.test.ts`

Cover:

- reading file-backed config,
- writing normalized config,
- browser/test fallback behavior,
- missing file behavior.

### Updated tests

- `src/tradingPanelStore.test.tsx`
- `src/tradingPanelApp.test.tsx`

Cover:

- demo card removal,
- config actions write through the file-backed helper,
- non-config state still persists to `localStorage`,
- legacy local config is used when file config is absent,
- file config overrides legacy local config when present.

## Verification

- `bunx vitest run src/globalConfigStorage.test.ts src/tradingPanelStore.test.tsx src/tradingPanelApp.test.tsx --exclude ".worktrees/**"`
- `npm run build`
- `cargo check`
- `bun run test` may still fail for pre-existing `.worktrees/**` discovery and unrelated failures; report separately if unchanged.
