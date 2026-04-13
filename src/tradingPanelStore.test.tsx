import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import * as globalConfigStorage from "./globalConfigStorage";
import { TRADING_PANEL_STORAGE_KEY, useTradingPanelState } from "./tradingPanelStore";

vi.mock("./globalConfigStorage", () => ({
  readGlobalConfigFallback: vi.fn(),
  readGlobalConfig: vi.fn(),
  writeGlobalConfig: vi.fn(),
}));

const readGlobalConfigFallbackMock = vi.mocked(globalConfigStorage.readGlobalConfigFallback);
const readGlobalConfigMock = vi.mocked(globalConfigStorage.readGlobalConfig);
const writeGlobalConfigMock = vi.mocked(globalConfigStorage.writeGlobalConfig);

function readPersistedState() {
  const raw = window.localStorage.getItem(TRADING_PANEL_STORAGE_KEY);
  return raw ? JSON.parse(raw) : null;
}

describe("useTradingPanelState preset actions", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.resetAllMocks();
    readGlobalConfigFallbackMock.mockReturnValue(null);
    readGlobalConfigMock.mockResolvedValue(null);
    writeGlobalConfigMock.mockResolvedValue();
  });

  it("writes config actions through the global config storage helper", async () => {
    const { result } = renderHook(() => useTradingPanelState());

    await waitFor(() => {
      expect(readGlobalConfigMock).toHaveBeenCalledTimes(1);
    });

    act(() => {
      result.current.actions.upsertPairPreset(" btcusdt ", "5m");
      result.current.actions.setDefaultPresetSymbol("BTCUSDT");
      result.current.actions.updateConfig({
        timerEnabled: true,
        timerInterval: 900,
      });
    });

    expect(result.current.state.pairPresets).toEqual([
      { symbol: "BTCUSDT", timeframe: "5m" },
    ]);
    expect(result.current.state.defaultPresetSymbol).toBe("BTCUSDT");
    expect(result.current.state.config).toEqual({
      timerEnabled: true,
      timerInterval: 900,
    });

    await waitFor(() => {
      expect(writeGlobalConfigMock).toHaveBeenLastCalledWith({
        config: {
          timerEnabled: true,
          timerInterval: 900,
        },
        pairPresets: [{ symbol: "BTCUSDT", timeframe: "5m" }],
        defaultPresetSymbol: "BTCUSDT",
      });
    });
  });

  it("persists config changes immediately enough for a fresh remount", async () => {
    let resolveWrite: (() => void) | undefined;
    writeGlobalConfigMock.mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          resolveWrite = resolve;
        }),
    );

    const { result } = renderHook(() => useTradingPanelState());

    await waitFor(() => {
      expect(readGlobalConfigMock).toHaveBeenCalledTimes(1);
    });

    act(() => {
      result.current.actions.upsertPairPreset("ethusdt", "15m");
      result.current.actions.setDefaultPresetSymbol("ETHUSDT");
    });

    expect(writeGlobalConfigMock).toHaveBeenCalledWith({
      config: {
        timerEnabled: false,
        timerInterval: 300,
      },
      pairPresets: [{ symbol: "ETHUSDT", timeframe: "15m" }],
      defaultPresetSymbol: "ETHUSDT",
    });

    resolveWrite?.();
  });

  it("does not write config during initial mount when hydration reads successfully", async () => {
    renderHook(() => useTradingPanelState());

    await waitFor(() => {
      expect(readGlobalConfigMock).toHaveBeenCalledTimes(1);
    });

    expect(writeGlobalConfigMock).not.toHaveBeenCalled();
  });

  it("clears the default preset when the preset is removed", () => {
    const { result } = renderHook(() => useTradingPanelState());

    act(() => {
      result.current.actions.upsertPairPreset("ETHUSDT", "1h");
      result.current.actions.setDefaultPresetSymbol("ethusdt");
    });

    expect(result.current.state.defaultPresetSymbol).toBe("ETHUSDT");

    act(() => {
      result.current.actions.removePairPreset(" ethusdt ");
    });

    expect(result.current.state.pairPresets).toEqual([]);
    expect(result.current.state.defaultPresetSymbol).toBeUndefined();
  });

  it("only accepts existing presets when setting the default symbol", () => {
    const { result } = renderHook(() => useTradingPanelState());

    act(() => {
      result.current.actions.setDefaultPresetSymbol("BTCUSDT");
    });

    expect(result.current.state.defaultPresetSymbol).toBeUndefined();

    act(() => {
      result.current.actions.upsertPairPreset("btcusdt", "5m");
      result.current.actions.setDefaultPresetSymbol(" btcusdt ");
    });

    expect(result.current.state.defaultPresetSymbol).toBe("BTCUSDT");
  });

  it("persists only session state to localStorage for non-config actions", async () => {
    const { result } = renderHook(() => useTradingPanelState());

    await waitFor(() => {
      expect(readGlobalConfigMock).toHaveBeenCalledTimes(1);
    });

    act(() => {
      result.current.actions.createJournal("ETHUSDT", "BINANCE");
      result.current.actions.updateChartView({
        symbol: "BINANCE:ETHUSDT",
        interval: "240",
      });
      result.current.actions.setPanelPinned(false);
    });

    await waitFor(() => {
      expect(readPersistedState()).toEqual({
        journals: [
          expect.objectContaining({
            symbol: "ETHUSDT",
            market: "BINANCE",
            status: "open",
          }),
        ],
        chart: {
          symbol: "BINANCE:ETHUSDT",
          interval: "240",
        },
        panelPinned: false,
      });
    });

    expect(readPersistedState()).not.toHaveProperty("config");
    expect(readPersistedState()).not.toHaveProperty("pairPresets");
    expect(readPersistedState()).not.toHaveProperty("defaultPresetSymbol");
    expect(writeGlobalConfigMock).not.toHaveBeenCalled();
  });

  it("uses legacy localStorage config when file config is absent", async () => {
    window.localStorage.setItem(
      TRADING_PANEL_STORAGE_KEY,
      JSON.stringify({
        journals: [],
        config: {
          timerEnabled: true,
          timerInterval: 120,
        },
        pairPresets: [{ symbol: "ethusdt", timeframe: "15m" }],
        defaultPresetSymbol: "ethusdt",
        chart: {
          symbol: "BINANCE:ETHUSDT",
          interval: "240",
        },
        panelPinned: false,
      }),
    );

    const { result } = renderHook(() => useTradingPanelState());

    await waitFor(() => {
      expect(result.current.state.config).toEqual({
        timerEnabled: true,
        timerInterval: 120,
      });
      expect(result.current.state.pairPresets).toEqual([
        { symbol: "ETHUSDT", timeframe: "15m" },
      ]);
      expect(result.current.state.defaultPresetSymbol).toBe("ETHUSDT");
    });
  });

  it("prefers file config over legacy localStorage config when available", async () => {
    window.localStorage.setItem(
      TRADING_PANEL_STORAGE_KEY,
      JSON.stringify({
        journals: [],
        config: {
          timerEnabled: false,
          timerInterval: 120,
        },
        pairPresets: [{ symbol: "btcusdt", timeframe: "5m" }],
        defaultPresetSymbol: "btcusdt",
        chart: {
          symbol: "BINANCE:ETHUSDT",
          interval: "240",
        },
        panelPinned: false,
      }),
    );

    readGlobalConfigMock.mockResolvedValue({
      config: {
        timerEnabled: true,
        timerInterval: 900,
      },
      pairPresets: [{ symbol: "ETHUSDT", timeframe: "1h" }],
      defaultPresetSymbol: "ETHUSDT",
    });

    const { result } = renderHook(() => useTradingPanelState());

    await waitFor(() => {
      expect(result.current.state.config).toEqual({
        timerEnabled: true,
        timerInterval: 900,
      });
      expect(result.current.state.pairPresets).toEqual([
        { symbol: "ETHUSDT", timeframe: "1h" },
      ]);
      expect(result.current.state.defaultPresetSymbol).toBe("ETHUSDT");
    });
  });

  it("does not write config when initial file hydration fails", async () => {
    readGlobalConfigMock.mockRejectedValue(new Error("fs read failed"));

    renderHook(() => useTradingPanelState());

    await waitFor(() => {
      expect(readGlobalConfigMock).toHaveBeenCalledTimes(1);
    });

    expect(writeGlobalConfigMock).not.toHaveBeenCalled();
  });

  it("does not let delayed file hydration overwrite user config edits", async () => {
    let resolveRead: ((value: Awaited<ReturnType<typeof globalConfigStorage.readGlobalConfig>>) => void) | undefined;
    readGlobalConfigMock.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveRead = resolve;
        }),
    );

    const { result } = renderHook(() => useTradingPanelState());

    act(() => {
      result.current.actions.upsertPairPreset("btcusdt", "5m");
      result.current.actions.setDefaultPresetSymbol("btcusdt");
      result.current.actions.updateConfig({
        timerEnabled: true,
        timerInterval: 900,
      });
    });

    act(() => {
      resolveRead?.({
        config: {
          timerEnabled: false,
          timerInterval: 120,
        },
        pairPresets: [{ symbol: "ETHUSDT", timeframe: "1h" }],
        defaultPresetSymbol: "ETHUSDT",
      });
    });

    await waitFor(() => {
      expect(result.current.state.config).toEqual({
        timerEnabled: true,
        timerInterval: 900,
      });
    });

    expect(result.current.state.pairPresets).toEqual([
      { symbol: "BTCUSDT", timeframe: "5m" },
    ]);
    expect(result.current.state.defaultPresetSymbol).toBe("BTCUSDT");
  });
});
