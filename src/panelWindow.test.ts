import { describe, expect, it, vi } from "vitest";

import {
  buildPanelUrl,
  openPanelWindow,
  parseAppView,
  readStoredPanelBounds,
} from "./panelWindow";

describe("parseAppView", () => {
  it("uses the main view by default", () => {
    expect(parseAppView("")).toEqual({ view: "main" });
  });

  it("detects the panel view from the query string", () => {
    expect(parseAppView("?view=panel")).toEqual({ view: "panel" });
  });
});

describe("buildPanelUrl", () => {
  it("creates the panel route", () => {
    expect(buildPanelUrl()).toBe("/?view=panel");
  });
});

describe("readStoredPanelBounds", () => {
  it("returns undefined when no layout has been saved", () => {
    expect(readStoredPanelBounds(null)).toBeUndefined();
  });

  it("parses saved bounds for panel restoration", () => {
    expect(
      readStoredPanelBounds(
        JSON.stringify({ x: 1200, y: 240, width: 392, height: 640 }),
      ),
    ).toEqual({ x: 1200, y: 240, width: 392, height: 640 });
  });
});

describe("openPanelWindow", () => {
  it("focuses an existing panel window instead of creating another one", async () => {
    const show = vi.fn(async () => undefined);
    const setFocus = vi.fn(async () => undefined);
    const getByLabel = vi.fn(async () => ({ show, setFocus }));
    const createWindow = vi.fn();

    await openPanelWindow({
      windowApi: { getByLabel, createWindow },
      storageApi: { getItem: () => null },
    });

    expect(getByLabel).toHaveBeenCalledWith("trading-panel");
    expect(show).toHaveBeenCalledOnce();
    expect(setFocus).toHaveBeenCalledOnce();
    expect(createWindow).not.toHaveBeenCalled();
  });

  it("creates the panel window with restored bounds when needed", async () => {
    const getByLabel = vi.fn(async () => null);
    const createWindow = vi.fn();

    await openPanelWindow({
      windowApi: { getByLabel, createWindow },
      storageApi: {
        getItem: () => JSON.stringify({ x: 1000, y: 180, width: 420, height: 700 }),
      },
    });

    expect(createWindow).toHaveBeenCalledWith("trading-panel", {
      title: "Trading Panel",
      url: "/?view=panel",
      width: 420,
      height: 700,
      x: 1000,
      y: 180,
      center: false,
      resizable: true,
      focus: true,
    });
  });
});
