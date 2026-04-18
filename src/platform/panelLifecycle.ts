import { WebviewWindow, getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";

import { PANEL_WINDOW_LABEL } from "./panelWindow";

export async function closePanelWindow(): Promise<void> {
  try {
    const panelWindow = await WebviewWindow.getByLabel(PANEL_WINDOW_LABEL);
    if (panelWindow) {
      await panelWindow.close();
      return;
    }
  } catch {
    // Fall through to current window and browser fallback.
  }

  try {
    const currentWindow = getCurrentWebviewWindow();
    await currentWindow.close();
    return;
  } catch {
    // Fall back for browser popup mode where no Tauri window exists.
  }

  if (typeof window !== "undefined") {
    window.close();
  }
}
