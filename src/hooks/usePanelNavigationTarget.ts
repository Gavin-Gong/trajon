import { getCurrentWindow } from "@tauri-apps/api/window";
import { useEffect, useMemo, useState } from "react";

import { currentJournal as selectCurrentJournal, type TradeJournal } from "../domain/journal";
import type { TradingJournalStore } from "../components/JournalStoreTypes";
import { PANEL_NAVIGATE_EVENT } from "../platform/panelWindow";

export function usePanelNavigationTarget(
  store: TradingJournalStore,
  presetSymbol?: string,
  journalId?: string,
): {
  journal: TradeJournal | undefined;
  targetPresetSymbol?: string;
} {
  const { state, actions } = store;
  const [target, setTarget] = useState({ presetSymbol, journalId });

  useEffect(() => {
    setTarget({ presetSymbol, journalId });
  }, [journalId, presetSymbol]);

  useEffect(() => {
    if (!(typeof window !== "undefined" && "__TAURI_INTERNALS__" in window)) {
      return;
    }

    const currentWindow = getCurrentWindow();
    let cancelled = false;
    let unlisten: (() => void) | undefined;

    void currentWindow
      .listen<{ presetSymbol?: string; journalId?: string }>(PANEL_NAVIGATE_EVENT, ({ payload }) => {
        if (!payload || cancelled) {
          return;
        }

        setTarget({
          presetSymbol: payload.presetSymbol,
          journalId: payload.journalId,
        });
        void actions.refreshState();
      })
      .then((dispose) => {
        unlisten = dispose;
      })
      .catch(() => undefined);

    return () => {
      cancelled = true;
      unlisten?.();
    };
  }, [actions]);

  const journal = useMemo(
    () =>
      (target.journalId
        ? state.journals.find((item) => item.id === target.journalId)
        : undefined) ??
      (target.presetSymbol
        ? state.journals.find((item) => item.status === "open" && item.symbol === target.presetSymbol)
        : undefined) ??
      selectCurrentJournal(state.journals),
    [state.journals, target.journalId, target.presetSymbol],
  );

  return {
    journal,
    targetPresetSymbol: target.presetSymbol,
  };
}
