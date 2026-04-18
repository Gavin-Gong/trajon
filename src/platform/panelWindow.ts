import { WebviewWindow } from "@tauri-apps/api/webviewWindow";

import { PANEL_BOUNDS_STORAGE_KEY, type PanelBounds } from "../model/storage";

type ExistingWindow = {
  emit?: <T>(event: string, payload?: T) => Promise<void>;
  show: () => Promise<void>;
  setFocus: () => Promise<void>;
};

type CreateWindowOptions = {
  title: string;
  url: string;
  width: number;
  maxWidth?: number;
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

type AppView = { view: "main" } | { view: "panel"; presetSymbol?: string; journalId?: string };
type PanelNavigationPayload = {
  presetSymbol?: string;
  journalId?: string;
};

export const PANEL_WINDOW_LABEL = "trading-panel";
export const PANEL_NAVIGATE_EVENT = "trading-panel:navigate";
export { PANEL_BOUNDS_STORAGE_KEY, type PanelBounds } from "../model/storage";
const PANEL_MAX_WIDTH = 400;

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
  return params.get("view") === "panel"
    ? {
        view: "panel",
        presetSymbol: params.get("presetSymbol") ?? undefined,
        journalId: params.get("journalId") ?? undefined,
      }
    : { view: "main" };
}

export function buildPanelUrl(options: PanelNavigationPayload = {}): string {
  const params = new URLSearchParams({ view: "panel" });

  if (options.presetSymbol) {
    params.set("presetSymbol", options.presetSymbol);
  }

  if (options.journalId) {
    params.set("journalId", options.journalId);
  }

  return `/?${params.toString()}`;
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
    presetSymbol?: string;
    journalId?: string;
    windowApi?: WindowApi;
    storageApi?: StorageApi;
  } = {},
): Promise<void> {
  if (!options.windowApi && !isTauriRuntime()) {
    const popup = window.open(
      buildPanelUrl({ presetSymbol: options.presetSymbol, journalId: options.journalId }),
      PANEL_WINDOW_LABEL,
      "popup=yes,width=300,height=640,left=980,top=180",
    );
    popup?.focus();
    return;
  }

  const windowApi = options.windowApi ?? defaultWindowApi;
  const storageApi = options.storageApi ?? defaultStorageApi;
  const existingWindow = await windowApi.getByLabel(PANEL_WINDOW_LABEL);

  if (existingWindow) {
    if (existingWindow.emit && (options.presetSymbol || options.journalId)) {
      await existingWindow.emit<PanelNavigationPayload>(PANEL_NAVIGATE_EVENT, {
        presetSymbol: options.presetSymbol,
        journalId: options.journalId,
      });
    }
    await existingWindow.show();
    await existingWindow.setFocus();
    return;
  }

  const bounds = readStoredPanelBounds(
    storageApi.getItem(PANEL_BOUNDS_STORAGE_KEY),
  );

  windowApi.createWindow(PANEL_WINDOW_LABEL, {
    title: "Trajon",
    url: buildPanelUrl({ presetSymbol: options.presetSymbol, journalId: options.journalId }),
    width: Math.min(bounds?.width ?? 300, PANEL_MAX_WIDTH),
    maxWidth: PANEL_MAX_WIDTH,
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
