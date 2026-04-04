import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";

type CurrentWindow = {
  isAlwaysOnTop: () => Promise<boolean>;
  setAlwaysOnTop: (alwaysOnTop: boolean) => Promise<void>;
};

type CurrentWindowApi = {
  getCurrentWindow: () => CurrentWindow;
};

const defaultCurrentWindowApi: CurrentWindowApi = {
  getCurrentWindow: () => getCurrentWebviewWindow(),
};

export async function readTopmostState(
  currentWindowApi: CurrentWindowApi = defaultCurrentWindowApi,
): Promise<boolean> {
  return currentWindowApi.getCurrentWindow().isAlwaysOnTop();
}

export async function toggleTopmostState(
  currentState: boolean,
  currentWindowApi: CurrentWindowApi = defaultCurrentWindowApi,
): Promise<boolean> {
  const nextState = !currentState;
  await currentWindowApi.getCurrentWindow().setAlwaysOnTop(nextState);
  return nextState;
}
