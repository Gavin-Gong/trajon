import type { TradeJournal } from "./model";
import type { GlobalConfig } from "./globalConfig";
import type { TradingPairPreset } from "./tradingPairPreset";

export type ChartViewState = {
  symbol: string;
  interval: string;
};

export type TradingPanelState = {
  journals: TradeJournal[];
  config: GlobalConfig;
  pairPresets: TradingPairPreset[];
  defaultPresetSymbol?: string;
  chart: ChartViewState;
  panelPinned: boolean;
};

export type PanelBounds = {
  x: number;
  y: number;
  width: number;
  height: number;
};

export const TRADING_PANEL_STORAGE_KEY = "trading-flow:trading-panel-state";
export const PANEL_BOUNDS_STORAGE_KEY = "trading-panel-bounds";

export type { GlobalConfig } from "./globalConfig";
export type StorageSchema = {
  [TRADING_PANEL_STORAGE_KEY]: TradingPanelState;
  [PANEL_BOUNDS_STORAGE_KEY]: PanelBounds;
};
