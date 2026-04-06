import { describe, expect, it } from "vitest";

import type { TimeframePreset, TradingPairPreset } from "./tradingPairPreset";
import type { TradingPanelState } from "./storage";
import {
  PANEL_BOUNDS_STORAGE_KEY,
  TRADING_PANEL_STORAGE_KEY,
} from "./storage";

describe("storage model exports", () => {
  it("defines the persisted storage keys in the model layer", () => {
    expect(TRADING_PANEL_STORAGE_KEY).toBe("trading-flow:trading-panel-state");
    expect(PANEL_BOUNDS_STORAGE_KEY).toBe("trading-panel-bounds");
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
      panelPinned: true,
    };

    expect(state.pairPresets[0]).toEqual({ symbol: "BTCUSDT", timeframe: "5m" });
    expect(state.defaultPresetSymbol).toBe("BTCUSDT");
  });
});
