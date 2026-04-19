import { useEffect, useMemo } from "react";

import type { TradingJournalStore } from "../components/JournalStoreTypes";
import type { TradeJournal } from "../domain/journal";

export function usePanelAutoCreateJournal(
  store: TradingJournalStore,
  journal: TradeJournal | undefined,
  presetSymbol?: string,
) {
  const { state, actions } = store;

  const defaultPreset = useMemo(() => {
    const incomingPreset = presetSymbol
      ? state.pairPresets.find((preset) => preset.symbol === presetSymbol)
      : undefined;

    return incomingPreset ??
      (state.defaultPresetSymbol
        ? state.pairPresets.find((preset) => preset.symbol === state.defaultPresetSymbol)
        : undefined);
  }, [presetSymbol, state.defaultPresetSymbol, state.pairPresets]);

  useEffect(() => {
    if (journal || !defaultPreset?.symbol) {
      return;
    }

    void actions.createJournal(defaultPreset.symbol, "");
  }, [actions, defaultPreset?.symbol, journal]);
}
