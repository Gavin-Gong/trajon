import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { TRADING_PANEL_STORAGE_KEY, useTradingPanelState } from "./tradingPanelStore";

describe("useTradingPanelState preset actions", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("upserts presets by normalized symbol instead of duplicating them", async () => {
    const { result } = renderHook(() => useTradingPanelState());

    act(() => {
      result.current.actions.upsertPairPreset(" btcusdt ", "5m");
    });

    act(() => {
      result.current.actions.upsertPairPreset("BTCUSDT", "15m");
    });

    expect(result.current.state.pairPresets).toEqual([
      { symbol: "BTCUSDT", timeframe: "15m" },
    ]);

    await waitFor(() => {
      expect(window.localStorage.getItem(TRADING_PANEL_STORAGE_KEY)).toContain("BTCUSDT");
    });
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

  it("updates and persists chart view state", async () => {
    const { result } = renderHook(() => useTradingPanelState());

    act(() => {
      result.current.actions.updateChartView({
        symbol: "BINANCE:ETHUSDT",
        interval: "240",
      });
    });

    expect(result.current.state.chart).toEqual({
      symbol: "BINANCE:ETHUSDT",
      interval: "240",
    });

    await waitFor(() => {
      expect(window.localStorage.getItem(TRADING_PANEL_STORAGE_KEY)).toContain("BINANCE:ETHUSDT");
    });
  });
});
