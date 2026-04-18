import { describe, expect, it, vi } from "vitest";
import defaultCapability from "../../../src-tauri/capabilities/default.json";

import {
  buildPanelUrl,
  openPanelWindow,
  parseAppView,
  readStoredPanelBounds,
} from "../panelWindow";

describe("parseAppView", () => {
  it("uses the main view by default", () => {
    expect(parseAppView("")).toEqual({ view: "main" });
  });

  it("detects the panel view from the query string", () => {
    expect(parseAppView("?view=panel")).toEqual({ view: "panel" });
  });

  it("reads presetSymbol and journalId from the panel query string", () => {
    expect(parseAppView("?view=panel&presetSymbol=ETHUSDT&journalId=journal-42")).toEqual({
      view: "panel",
      presetSymbol: "ETHUSDT",
      journalId: "journal-42",
    });
  });
});

describe("buildPanelUrl", () => {
  it("creates the panel route", () => {
    expect(buildPanelUrl()).toBe("/?view=panel");
  });

  it("includes presetSymbol and journalId when provided", () => {
    expect(buildPanelUrl({ presetSymbol: "ETHUSDT", journalId: "journal-42" })).toBe(
      "/?view=panel&presetSymbol=ETHUSDT&journalId=journal-42",
    );
  });
});

describe("readStoredPanelBounds", () => {
  it("returns undefined when no layout has been saved", () => {
    expect(readStoredPanelBounds(null)).toBeUndefined();
  });

  it("parses saved bounds for panel restoration", () => {
    expect(
      readStoredPanelBounds(
        JSON.stringify({ x: 1200, y: 240, width: 300, height: 640 }),
      ),
    ).toEqual({ x: 1200, y: 240, width: 300, height: 640 });
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

  it("retargets an existing panel window before focusing it", async () => {
    const emit = vi.fn(async () => undefined);
    const show = vi.fn(async () => undefined);
    const setFocus = vi.fn(async () => undefined);
    const getByLabel = vi.fn(async () => ({ emit, show, setFocus }));
    const createWindow = vi.fn();

    await openPanelWindow({
      journalId: "journal-42",
      windowApi: { getByLabel, createWindow },
      storageApi: { getItem: () => null },
    });

    expect(emit).toHaveBeenCalledWith("trading-panel:navigate", {
      journalId: "journal-42",
      presetSymbol: undefined,
    });
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
      title: "Trajon",
      url: "/?view=panel",
      width: 400,
      maxWidth: 400,
      height: 700,
      x: 1000,
      y: 180,
      center: false,
      resizable: true,
      focus: true,
    });
  });

  it("uses a 300px default width when no saved bounds exist", async () => {
    const getByLabel = vi.fn(async () => null);
    const createWindow = vi.fn();

    await openPanelWindow({
      windowApi: { getByLabel, createWindow },
      storageApi: { getItem: () => null },
    });

    expect(createWindow).toHaveBeenCalledWith("trading-panel", {
      title: "Trajon",
      url: "/?view=panel",
      width: 300,
      maxWidth: 400,
      height: 640,
      x: undefined,
      y: undefined,
      center: true,
      resizable: true,
      focus: true,
    });
  });

  it("passes journalId into the created panel url", async () => {
    const getByLabel = vi.fn(async () => null);
    const createWindow = vi.fn();

    await openPanelWindow({
      journalId: "journal-42",
      windowApi: { getByLabel, createWindow },
      storageApi: { getItem: () => null },
    });

    expect(createWindow).toHaveBeenCalledWith("trading-panel", {
      title: "Trajon",
      url: "/?view=panel&journalId=journal-42",
      width: 300,
      maxWidth: 400,
      height: 640,
      x: undefined,
      y: undefined,
      center: true,
      resizable: true,
      focus: true,
    });
  });

  it("grants the trading panel window permission to toggle always-on-top", () => {
    const capability = defaultCapability as {
      windows?: string[];
      permissions?: string[];
    };

    expect(capability.windows).toContain("trading-panel");
    expect(capability.permissions).toContain("core:window:allow-set-always-on-top");
  });
});
