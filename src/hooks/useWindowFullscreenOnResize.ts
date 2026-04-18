import { getCurrentWindow } from "@tauri-apps/api/window";
import { useEffect, useState } from "react";

export function useWindowFullscreenOnResize() {
  const [isFullscreen, setIsFullscreen] = useState(false);

  useEffect(() => {
    if (!isTauriRuntime()) {
      return;
    }

    const currentWindow = getCurrentWindow();
    let disposed = false;

    const syncFullscreenState = async () => {
      const nextIsFullscreen = await currentWindow.isFullscreen();
      if (!disposed) {
        setIsFullscreen(nextIsFullscreen);
      }
    };

    const handleResize = () => {
      void syncFullscreenState().catch(() => undefined);
    };

    void syncFullscreenState().catch(() => undefined);
    window.addEventListener("resize", handleResize);

    return () => {
      disposed = true;
      window.removeEventListener("resize", handleResize);
    };
  }, []);

  return isFullscreen;
}

function isTauriRuntime(): boolean {
  return typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;
}
