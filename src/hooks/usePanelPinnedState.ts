import { getCurrentWindow } from "@tauri-apps/api/window";
import { useEffect, useState } from "react";

import type { TradingJournalStore } from "../components/JournalStoreTypes";

export function usePanelPinnedState(store: TradingJournalStore) {
  const { state, actions } = store;
  const [isPinned, setIsPinned] = useState(state.panelPinned);

  useEffect(() => {
    setIsPinned(state.panelPinned);
  }, [state.panelPinned]);

  useEffect(() => {
    if (!(typeof window !== "undefined" && "__TAURI_INTERNALS__" in window)) {
      return;
    }

    const currentWindow = getCurrentWindow();
    void currentWindow.setAlwaysOnTop(isPinned).catch(() => undefined);
  }, [isPinned]);

  const togglePinned = () => {
    const nextPinned = !isPinned;
    setIsPinned(nextPinned);
    void actions.setPanelPinned(nextPinned);
  };

  return { isPinned, togglePinned };
}
