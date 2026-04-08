import { describe, expect, it } from "vitest";

import type { GlobalConfig } from "./globalConfig";
import {
  PANEL_BOUNDS_STORAGE_KEY,
  TRADING_PANEL_STORAGE_KEY,
} from "./storage";

describe("storage model exports", () => {
  it("defines the persisted storage keys in the model layer", () => {
    expect(TRADING_PANEL_STORAGE_KEY).toBe("trading-flow:trading-panel-state");
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
});
