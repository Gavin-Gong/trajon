import { getCurrentWindow } from "@tauri-apps/api/window";
import { useEffect } from "react";

import { PANEL_BOUNDS_STORAGE_KEY } from "../model/storage";
import { serializePanelBounds } from "../platform/panelWindow";

export function usePersistPanelBounds() {
  useEffect(() => {
    if (!isTauriRuntime()) {
      return;
    }

    const currentWindow = getCurrentWindow();
    const persistBounds = async () => {
      const [position, size] = await Promise.all([
        currentWindow.outerPosition(),
        currentWindow.outerSize(),
      ]);

      window.localStorage.setItem(
        PANEL_BOUNDS_STORAGE_KEY,
        serializePanelBounds({
          x: position.x,
          y: position.y,
          width: size.width,
          height: size.height,
        }),
      );
    };

    void persistBounds().catch(() => undefined);
    const unsubscribers: Array<() => void> = [];

    void currentWindow
      .onMoved(() => {
        void persistBounds().catch(() => undefined);
      })
      .then((unlisten) => {
        unsubscribers.push(unlisten);
      });

    void currentWindow
      .onResized(() => {
        void persistBounds().catch(() => undefined);
      })
      .then((unlisten) => {
        unsubscribers.push(unlisten);
      });

    return () => {
      unsubscribers.forEach((unlisten) => unlisten());
    };
  }, []);
}

function isTauriRuntime(): boolean {
  return typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;
}
