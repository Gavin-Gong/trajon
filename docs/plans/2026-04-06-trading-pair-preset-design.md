# Trading Pair Preset Design

**Goal:** Add configurable trading-pair presets so users can store a default timeframe per symbol, choose one default preset, and automatically prefill that preset only when creating a new journal.

**Current Context:**
- The app stores `TradingPanelState` in local storage under `trading-flow:trading-panel-state`.
- Global reminder settings already live in `state.config`.
- The settings page currently manages only reminder-related options.
- The trading panel creates journals from user-entered symbol and market values.

## Requirements

1. Users can maintain a list of trading-pair presets.
2. Each preset contains a `symbol` and its default `timeframe`.
3. Users can set one preset as the default preset.
4. The default preset is applied only when creating a new journal.
5. Manual edits after creation are not overridden by the preset.
6. Presets are persisted in local storage with the rest of trading panel state.

## Chosen Model

Keep reminder settings inside `GlobalConfig`, and store trading-pair presets as separate top-level business data in `TradingPanelState`.

```ts
export type TimeframePreset =
  | "1m"
  | "3m"
  | "5m"
  | "15m"
  | "30m"
  | "1h"
  | "4h"
  | "1d";

export type TradingPairPreset = {
  symbol: string;
  timeframe: TimeframePreset;
};

export type GlobalConfig = {
  timerEnabled: boolean;
  timerInterval: number;
};

export type TradingPanelState = {
  journals: TradeJournal[];
  config: GlobalConfig;
  pairPresets: TradingPairPreset[];
  defaultPresetSymbol?: string;
  panelPinned: boolean;
};
```

## Why This Model

### Recommended approach

Store `pairPresets` and `defaultPresetSymbol` directly on `TradingPanelState` instead of placing them inside `GlobalConfig`.

Why:
- `GlobalConfig` remains focused on true global app settings.
- Trading-pair presets are business preference data, not general UI configuration.
- Future preset expansion can stay isolated from reminder settings.
- This avoids turning `GlobalConfig` into a mixed settings bucket.

### Alternatives considered

1. Add presets directly to `GlobalConfig`
- Lowest implementation cost.
- Rejected because it mixes reminder settings and trading defaults into one model.

2. Nest settings into `config.reminder` and `config.presets`
- Clear grouping.
- Rejected for now because it would trigger a broader settings refactor than this feature needs.

## Validation Rules

1. `symbol`
- Trim whitespace before storing.
- Normalize to uppercase before storing.
- Must be unique within `pairPresets`.

2. `timeframe`
- Must be one of the allowed preset values.

3. `defaultPresetSymbol`
- Must match an existing preset `symbol`.
- If the referenced preset is deleted or invalid during rehydrate, clear the default.

## Storage Behavior

The persisted trading panel state will gain two new top-level fields:
- `pairPresets`
- `defaultPresetSymbol`

Default state:

```ts
{
  journals: [],
  config: {
    timerEnabled: false,
    timerInterval: 300,
  },
  pairPresets: [],
  defaultPresetSymbol: undefined,
  panelPinned: true,
}
```

Normalization behavior:
- Non-array `pairPresets` falls back to `[]`.
- Invalid preset rows are dropped individually.
- Invalid `defaultPresetSymbol` is cleared.
- Existing reminder settings continue to normalize exactly as they do now.

## UI Behavior

The settings screen gains a dedicated trading-pair presets section alongside the existing reminder settings.

Supported actions:
1. Add a preset with `symbol` and `timeframe`.
2. Update an existing preset by symbol.
3. Delete a preset.
4. Mark one preset as the default preset.

Expected interaction rules:
- Saving a preset for an existing symbol updates its timeframe instead of creating a duplicate.
- Deleting the current default preset clears `defaultPresetSymbol`.
- If no default preset is set, journal creation falls back to current behavior.

## Journal Creation Behavior

Default preset application happens only at journal creation time.

Creation flow:
1. User starts creating a new journal.
2. If `defaultPresetSymbol` exists, resolve that preset from `pairPresets`.
3. Prefill the symbol from the preset.
4. Prefill the preset timeframe for the creation flow.
5. After the journal is created, the user can edit values manually without further preset enforcement.

Non-goals for this feature:
- No continuous syncing between active journal and preset.
- No automatic overwriting of manual edits.
- No multi-default support.

## Error Handling

1. Duplicate symbol save
- Update the existing preset instead of creating a duplicate record.

2. Invalid stored data
- Drop invalid presets and continue loading the rest of the state.

3. Missing default target
- Clear `defaultPresetSymbol` rather than failing the whole state restore.

## Testing Scope

Implementation should verify:
1. Storage normalization for valid and invalid preset rows.
2. Clearing invalid `defaultPresetSymbol` during rehydrate.
3. Updating an existing preset by symbol.
4. Clearing default when its preset is deleted.
5. Applying the default preset only during journal creation.
6. Preserving existing reminder settings behavior.

## Summary

This design adds trading-pair presets as separate persisted business data on `TradingPanelState`, keeps `GlobalConfig` focused on reminder settings, and introduces a single default preset that is applied only when creating a new journal. This gives the feature room to grow without forcing a broader settings refactor now.
