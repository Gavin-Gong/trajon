import { WebviewWindow } from "@tauri-apps/api/webviewWindow";

type ExistingWindow = {
  show: () => Promise<void>;
  setFocus: () => Promise<void>;
};

type CreateWindowOptions = {
  title: string;
  url: string;
  width: number;
  height: number;
  x?: number;
  y?: number;
  center: boolean;
  resizable: boolean;
  focus: boolean;
};

type WindowApi = {
  getByLabel: (label: string) => Promise<ExistingWindow | null>;
  createWindow: (label: string, options: CreateWindowOptions) => void;
};

type StorageApi = {
  getItem: (key: string) => string | null;
};

type AppView = { view: "main" } | { view: "panel" };

export type PanelBounds = {
  x: number;
  y: number;
  width: number;
  height: number;
};

export const PANEL_WINDOW_LABEL = "trading-panel";
export const PANEL_BOUNDS_STORAGE_KEY = "trading-panel-bounds";

const defaultWindowApi: WindowApi = {
  getByLabel: (label) => WebviewWindow.getByLabel(label),
  createWindow: (label, options) => {
    new WebviewWindow(label, options);
  },
};

const defaultStorageApi: StorageApi = {
  getItem: (key) => {
    if (typeof window === "undefined") {
      return null;
    }

    return window.localStorage.getItem(key);
  },
};

export function parseAppView(search: string): AppView {
  const params = new URLSearchParams(search);
  return params.get("view") === "panel" ? { view: "panel" } : { view: "main" };
}

export function buildPanelUrl(): string {
  return "/?view=panel";
}

export function readStoredPanelBounds(rawValue: string | null): PanelBounds | undefined {
  if (!rawValue) {
    return undefined;
  }

  try {
    const parsed = JSON.parse(rawValue) as Partial<PanelBounds>;

    if (
      typeof parsed.x !== "number" ||
      typeof parsed.y !== "number" ||
      typeof parsed.width !== "number" ||
      typeof parsed.height !== "number"
    ) {
      return undefined;
    }

    return {
      x: parsed.x,
      y: parsed.y,
      width: parsed.width,
      height: parsed.height,
    };
  } catch {
    return undefined;
  }
}

export function serializePanelBounds(bounds: PanelBounds): string {
  return JSON.stringify(bounds);
}

export async function openPanelWindow(
  options: {
    windowApi?: WindowApi;
    storageApi?: StorageApi;
  } = {},
): Promise<void> {
  if (!options.windowApi && !isTauriRuntime()) {
    const popup = window.open(
      buildPanelUrl(),
      PANEL_WINDOW_LABEL,
      "popup=yes,width=392,height=640,left=980,top=180",
    );
    popup?.focus();
    return;
  }

  const windowApi = options.windowApi ?? defaultWindowApi;
  const storageApi = options.storageApi ?? defaultStorageApi;
  const existingWindow = await windowApi.getByLabel(PANEL_WINDOW_LABEL);

  if (existingWindow) {
    await existingWindow.show();
    await existingWindow.setFocus();
    return;
  }

  const bounds = readStoredPanelBounds(
    storageApi.getItem(PANEL_BOUNDS_STORAGE_KEY),
  );

  windowApi.createWindow(PANEL_WINDOW_LABEL, {
    title: "Trading Panel",
    url: buildPanelUrl(),
    width: bounds?.width ?? 392,
    height: bounds?.height ?? 640,
    x: bounds?.x,
    y: bounds?.y,
    center: bounds === undefined,
    resizable: true,
    focus: true,
  });
}

function isTauriRuntime(): boolean {
  return typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;
}
