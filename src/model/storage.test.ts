import { describe, expect, it } from "vitest";

import type { GlobalConfig } from "./globalConfig";
import type { TimeframePreset, TradingPairPreset } from "./tradingPairPreset";
import type { TradingPanelState } from "./storage";
import { PANEL_BOUNDS_STORAGE_KEY } from "./storage";

describe("storage model exports", () => {
  it("defines the persisted storage keys in the model layer", () => {
    expect(PANEL_BOUNDS_STORAGE_KEY).toBe("trading-panel-bounds");
  });

  it("keeps global config in a dedicated model module", () => {
    const config: GlobalConfig = {
      timerEnabled: true,
      timerInterval: 300,
    };

    expect(config).toEqual({
      timerEnabled: true,
      timerInterval: 300,
    });
  });

  it("exposes trading pair preset models through the model layer", () => {
    const timeframe: TimeframePreset = "5m";
    const preset: TradingPairPreset = {
      symbol: "BTCUSDT",
      timeframe,
    };
    const state: TradingPanelState = {
      journals: [],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [preset],
      defaultPresetSymbol: "BTCUSDT",
      chart: {
        symbol: "BINANCE:BTCUSDT",
        interval: "60",
      },
      panelPinned: true,
    };

    expect(state.pairPresets[0]).toEqual({ symbol: "BTCUSDT", timeframe: "5m" });
    expect(state.defaultPresetSymbol).toBe("BTCUSDT");
  });

  it("includes chart view state in persisted storage models", () => {
    const state: TradingPanelState = {
      journals: [],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      chart: {
        symbol: "BINANCE:BTCUSDT",
        interval: "60",
      },
      panelPinned: true,
    };

    expect(state.chart).toEqual({ symbol: "BINANCE:BTCUSDT", interval: "60" });
  });
});
