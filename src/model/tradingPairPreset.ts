export const TIMEFRAME_PRESET_VALUES = ["1m", "3m", "5m", "15m", "30m", "1h", "4h", "1d"] as const;

export type TimeframePreset = (typeof TIMEFRAME_PRESET_VALUES)[number];

export type TradingPairPreset = {
  symbol: string;
  timeframe: TimeframePreset;
};
