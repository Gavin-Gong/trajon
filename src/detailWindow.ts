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
  center: boolean;
  resizable: boolean;
};

type WindowApi = {
  getByLabel: (label: string) => Promise<ExistingWindow | null>;
  createWindow: (label: string, options: CreateWindowOptions) => void;
};

type ViewState =
  | { view: "main" }
  | {
      view: "detail";
      symbol: string;
    };

const defaultWindowApi: WindowApi = {
  getByLabel: (label) => WebviewWindow.getByLabel(label),
  createWindow: (label, options) => {
    new WebviewWindow(label, options);
  },
};

export function normalizeSymbol(symbol: string): string {
  return symbol.trim().replace(/\s+/g, "").toUpperCase();
}

export function createWindowLabel(symbol: string): string {
  const normalizedSymbol = symbol
    .trim()
    .toLowerCase()
    .replace(/\s+/g, "-")
    .replace(/[^a-z0-9\-/:_]+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "");

  return `detail-${normalizedSymbol}`;
}

export function buildDetailUrl(symbol: string): string {
  const params = new URLSearchParams({
    view: "detail",
    symbol: normalizeSymbol(symbol),
  });

  return `/?${params.toString()}`;
}

export function parseViewState(search: string): ViewState {
  const params = new URLSearchParams(search);

  if (params.get("view") !== "detail") {
    return { view: "main" };
  }

  const symbol = normalizeSymbol(params.get("symbol") ?? "");
  if (!symbol) {
    return { view: "main" };
  }

  return {
    view: "detail",
    symbol,
  };
}

export async function openDetailWindow(
  symbol: string,
  windowApi: WindowApi = defaultWindowApi,
): Promise<void> {
  const normalizedSymbol = normalizeSymbol(symbol);
  if (!normalizedSymbol) {
    return;
  }

  const label = createWindowLabel(normalizedSymbol);
  const existingWindow = await windowApi.getByLabel(label);

  if (existingWindow) {
    await existingWindow.show();
    await existingWindow.setFocus();
    return;
  }

  windowApi.createWindow(label, {
    title: `Detail - ${normalizedSymbol}`,
    url: buildDetailUrl(normalizedSymbol),
    width: 1100,
    height: 760,
    center: true,
    resizable: true,
  });
}
