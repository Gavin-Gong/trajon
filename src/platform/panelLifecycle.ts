import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";


export async function closePanelWindow(): Promise<void> {
  try {
    const currentWindow = getCurrentWebviewWindow();
    await currentWindow.close();
    return;
  } catch {
    // Fall back for browser popup mode where no Tauri window exists.
  }
}
