import { describe, expect, it, vi } from "vitest";

import {
  buildDetailUrl,
  createWindowLabel,
  normalizeSymbol,
  openDetailWindow,
  parseViewState,
} from "./detailWindow";

describe("normalizeSymbol", () => {
  it("trims and uppercases symbols", () => {
    expect(normalizeSymbol(" btcusdt ")).toBe("BTCUSDT");
  });

  it("removes inner whitespace", () => {
    expect(normalizeSymbol(" btc / usdt ")).toBe("BTC/USDT");
  });
});

describe("createWindowLabel", () => {
  it("creates a deterministic safe label", () => {
    expect(createWindowLabel("BTC/USDT perp")).toBe("detail-btc/usdt-perp");
  });
});

describe("buildDetailUrl", () => {
  it("encodes the detail view and symbol in the query string", () => {
    expect(buildDetailUrl("BTCUSDT")).toBe("/?view=detail&symbol=BTCUSDT");
  });
});

describe("parseViewState", () => {
  it("returns the main view when no detail params are present", () => {
    expect(parseViewState("")).toEqual({ view: "main" });
  });

  it("returns the detail view when a symbol is present", () => {
    expect(parseViewState("?view=detail&symbol=ethusdt")).toEqual({
      view: "detail",
      symbol: "ETHUSDT",
    });
  });

  it("falls back to main when the symbol is blank", () => {
    expect(parseViewState("?view=detail&symbol=   ")).toEqual({ view: "main" });
  });
});

describe("openDetailWindow", () => {
  it("focuses an existing window for the same symbol", async () => {
    const show = vi.fn(async () => undefined);
    const setFocus = vi.fn(async () => undefined);
    const createWindow = vi.fn();
    const getByLabel = vi.fn(async () => ({ show, setFocus }));

    await openDetailWindow(" btcusdt ", { getByLabel, createWindow });

    expect(getByLabel).toHaveBeenCalledWith("detail-btcusdt");
    expect(show).toHaveBeenCalledOnce();
    expect(setFocus).toHaveBeenCalledOnce();
    expect(createWindow).not.toHaveBeenCalled();
  });

  it("creates a new window for a new symbol", async () => {
    const createWindow = vi.fn();
    const getByLabel = vi.fn(async () => null);

    await openDetailWindow("ethusdt", { getByLabel, createWindow });

    expect(createWindow).toHaveBeenCalledWith("detail-ethusdt", {
      title: "Detail - ETHUSDT",
      url: "/?view=detail&symbol=ETHUSDT",
      width: 1100,
      height: 760,
      center: true,
      resizable: true,
    });
  });
});
