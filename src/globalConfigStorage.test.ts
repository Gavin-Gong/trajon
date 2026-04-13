import { beforeEach, describe, expect, it, vi } from "vitest";
import { BaseDirectory, exists, mkdir, readTextFile, writeTextFile } from "@tauri-apps/plugin-fs";

const fsMock = vi.hoisted(() => ({
  BaseDirectory: {
    Home: "Home",
  },
  exists: vi.fn(),
  mkdir: vi.fn(),
  readTextFile: vi.fn(),
  writeTextFile: vi.fn(),
}));

vi.mock("@tauri-apps/plugin-fs", () => fsMock);

import {
  GLOBAL_CONFIG_DIRECTORY,
  GLOBAL_CONFIG_FALLBACK_STORAGE_KEY,
  GLOBAL_CONFIG_FILE_PATH,
  readGlobalConfig,
  writeGlobalConfig,
} from "./globalConfigStorage";

declare global {
  interface Window {
    __TAURI_INTERNALS__?: unknown;
  }
}

describe("globalConfigStorage", () => {
  beforeEach(() => {
    delete window.__TAURI_INTERNALS__;
    window.localStorage.clear();
    vi.mocked(exists).mockReset();
    vi.mocked(mkdir).mockReset();
    vi.mocked(readTextFile).mockReset();
    vi.mocked(writeTextFile).mockReset();
  });

  it("reads stored config from the home-scoped config file", async () => {
    window.__TAURI_INTERNALS__ = {};
    vi.mocked(exists).mockResolvedValue(true);
    vi.mocked(readTextFile).mockResolvedValue(
      JSON.stringify({
        config: {
          timerEnabled: true,
          timerInterval: 120,
        },
        pairPresets: [
          { symbol: " btcusdt ", timeframe: "1h" },
          { symbol: "ethusdt", timeframe: "bad" },
        ],
        defaultPresetSymbol: " btcusdt ",
      }),
    );

    await expect(readGlobalConfig()).resolves.toEqual({
      config: {
        timerEnabled: true,
        timerInterval: 120,
      },
      pairPresets: [{ symbol: "BTCUSDT", timeframe: "1h" }],
      defaultPresetSymbol: "BTCUSDT",
    });
    expect(readTextFile).toHaveBeenCalledWith(GLOBAL_CONFIG_FILE_PATH, {
      baseDir: BaseDirectory.Home,
    });
  });

  it("normalizes invalid timerInterval values when reading persisted config", async () => {
    window.__TAURI_INTERNALS__ = {};
    vi.mocked(exists).mockResolvedValue(true);
    vi.mocked(readTextFile).mockResolvedValue(
      JSON.stringify({
        config: {
          timerEnabled: true,
          timerInterval: -5,
        },
        pairPresets: [],
      }),
    );

    await expect(readGlobalConfig()).resolves.toEqual({
      config: {
        timerEnabled: true,
        timerInterval: 300,
      },
      pairPresets: [],
      defaultPresetSymbol: undefined,
    });
  });

  it("returns fresh default config objects for repeated malformed reads", async () => {
    window.__TAURI_INTERNALS__ = {};
    vi.mocked(exists).mockResolvedValue(true);
    vi.mocked(readTextFile).mockResolvedValue(JSON.stringify({ config: "bad", pairPresets: [] }));

    const first = await readGlobalConfig();
    if (!first) {
      throw new Error("expected first config result");
    }

    first.config.timerInterval = 999;

    const second = await readGlobalConfig();
    expect(second).toEqual({
      config: {
        timerEnabled: false,
        timerInterval: 300,
      },
      pairPresets: [],
      defaultPresetSymbol: undefined,
    });
  });

  it("returns null when fallback storage contains malformed JSON", async () => {
    window.localStorage.setItem(GLOBAL_CONFIG_FALLBACK_STORAGE_KEY, "not-json");

    await expect(readGlobalConfig()).resolves.toBeNull();
  });

  it("returns null when config.json contains malformed JSON", async () => {
    window.__TAURI_INTERNALS__ = {};
    vi.mocked(exists).mockResolvedValue(true);
    vi.mocked(readTextFile).mockResolvedValue("not-json");

    await expect(readGlobalConfig()).resolves.toBeNull();
  });

  it("returns null when config.json disappears between exists and read", async () => {
    window.__TAURI_INTERNALS__ = {};
    vi.mocked(exists).mockResolvedValue(true);
    vi.mocked(readTextFile).mockRejectedValue(new Error("No such file or directory"));

    await expect(readGlobalConfig()).resolves.toBeNull();
  });

  it("rethrows unexpected fs read errors", async () => {
    window.__TAURI_INTERNALS__ = {};
    vi.mocked(exists).mockResolvedValue(true);
    vi.mocked(readTextFile).mockRejectedValue(new Error("permission denied"));

    await expect(readGlobalConfig()).rejects.toThrow("permission denied");
  });

  it("returns null when config.json is missing", async () => {
    window.__TAURI_INTERNALS__ = {};
    vi.mocked(exists).mockResolvedValue(false);

    await expect(readGlobalConfig()).resolves.toBeNull();
  });

  it("writes normalized config to config.json", async () => {
    window.__TAURI_INTERNALS__ = {};

    await writeGlobalConfig({
      config: {
        timerEnabled: true,
        timerInterval: Number.NaN,
      },
      pairPresets: [
        { symbol: " btcusdt ", timeframe: "1h" },
        { symbol: "", timeframe: "15m" },
      ],
      defaultPresetSymbol: " btcusdt ",
    });

    expect(mkdir).toHaveBeenCalledWith(GLOBAL_CONFIG_DIRECTORY, {
      baseDir: BaseDirectory.Home,
      recursive: true,
    });
    expect(writeTextFile).toHaveBeenCalledWith(
      GLOBAL_CONFIG_FILE_PATH,
      JSON.stringify({
        config: {
          timerEnabled: true,
          timerInterval: 300,
        },
        pairPresets: [{ symbol: "BTCUSDT", timeframe: "1h" }],
        defaultPresetSymbol: "BTCUSDT",
      }),
      { baseDir: BaseDirectory.Home },
    );
  });

  it("uses localStorage fallback when Tauri fs is unavailable", async () => {
    await writeGlobalConfig({
      config: {
        timerEnabled: true,
        timerInterval: 120,
      },
      pairPresets: [{ symbol: "btcusdt", timeframe: "1h" }],
      defaultPresetSymbol: "btcusdt",
    });

    await expect(
      readGlobalConfig(),
    ).resolves.toEqual({
      config: {
        timerEnabled: true,
        timerInterval: 120,
      },
      pairPresets: [{ symbol: "BTCUSDT", timeframe: "1h" }],
      defaultPresetSymbol: "BTCUSDT",
    });

    expect(window.localStorage.getItem(GLOBAL_CONFIG_FALLBACK_STORAGE_KEY)).toContain("BTCUSDT");
    expect(mkdir).not.toHaveBeenCalled();
    expect(writeTextFile).not.toHaveBeenCalled();
    expect(exists).not.toHaveBeenCalled();
    expect(readTextFile).not.toHaveBeenCalled();
  });

  it("returns null from fallback storage when no config is present", async () => {
    await expect(readGlobalConfig()).resolves.toBeNull();

    await expect(
      writeGlobalConfig({
        config: {
          timerEnabled: true,
          timerInterval: 120,
        },
        pairPresets: [{ symbol: "btcusdt", timeframe: "1h" }],
        defaultPresetSymbol: "btcusdt",
      }),
    ).resolves.toBeUndefined();
  });
});
