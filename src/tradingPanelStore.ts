import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import {
  composeTradeEvent,
  createJournal as createDomainJournal,
  type TradeEventComposerResult,
  type TradeEventDraft,
  type TradeJournal,
} from "./tradingPanelDomain";
import {
  TIMEFRAME_PRESET_VALUES,
  type TimeframePreset,
  type TradingPairPreset,
} from "./model/tradingPairPreset";
import {
  TRADING_PANEL_STORAGE_KEY,
  type ChartViewState,
  type TradingPanelState,
} from "./model/storage";
import type { GlobalConfig } from "./model/globalConfig";
import {
  readGlobalConfigFallback,
  readGlobalConfig,
  type StoredGlobalConfig,
  writeGlobalConfig,
} from "./globalConfigStorage";

export {
  TRADING_PANEL_STORAGE_KEY,
  type GlobalConfig,
  type TradingPanelState,
} from "./model/storage";

const defaultState: TradingPanelState = {
  journals: [],
  config: {
    timerEnabled: false,
    timerInterval: 300,
  },
  pairPresets: [],
  defaultPresetSymbol: undefined,
  chart: {
    symbol: "BINANCE:BTCUSDT",
    interval: "60",
  },
  panelPinned: true,
};

const TIMEFRAME_PRESETS = new Set<string>(TIMEFRAME_PRESET_VALUES);
const CHART_SYMBOLS = new Set<string>([
  "BINANCE:BTCUSDT",
  "BINANCE:ETHUSDT",
  "NASDAQ:AAPL",
  "SSE:600519",
  "CME_MINI:NQ1!",
  "CME_MINI:ES1!",
]);
const CHART_INTERVALS = new Set<string>(["1", "5", "15", "30", "60", "240", "1D"]);

type TradingPanelSessionState = Pick<TradingPanelState, "journals" | "chart" | "panelPinned">;
type ConfigHydrationStatus = "pending" | "ready" | "failed";

export function readTradingPanelState(): TradingPanelState {
  if (typeof window === "undefined") {
    return defaultState;
  }

  const raw = window.localStorage.getItem(TRADING_PANEL_STORAGE_KEY);
  const sessionState = raw
    ? (() => {
        try {
          return normalizeState(JSON.parse(raw));
        } catch {
          return defaultState;
        }
      })()
    : defaultState;

  const fallbackConfig = readGlobalConfigFallback();
  return fallbackConfig ? normalizeState({ ...sessionState, ...fallbackConfig }) : sessionState;
}

function writeTradingPanelSessionState(state: TradingPanelState) {
  if (typeof window === "undefined") {
    return;
  }

  window.localStorage.setItem(TRADING_PANEL_STORAGE_KEY, JSON.stringify(pickSessionState(state)));
}

export function useTradingPanelState() {
  const [state, setState] = useState<TradingPanelState>(() => readTradingPanelState());
  const [configHydrationStatus, setConfigHydrationStatus] = useState<ConfigHydrationStatus>("pending");
  const [configDirty, setConfigDirty] = useState(false);
  const configHydrationStatusRef = useRef<ConfigHydrationStatus>("pending");

  useEffect(() => {
    configHydrationStatusRef.current = configHydrationStatus;
  }, [configHydrationStatus]);

  useEffect(() => {
    let cancelled = false;

    void readGlobalConfig()
      .then((storedConfig) => {
        if (cancelled) {
          return;
        }

        if (storedConfig) {
          setState((currentState) => {
            if (configDirty) {
              return currentState;
            }

            return normalizeState({ ...currentState, ...storedConfig });
          });
        }

        setConfigHydrationStatus("ready");
      })
      .catch(() => {
        if (!cancelled) {
          setConfigHydrationStatus("failed");
        }
      });

    return () => {
      cancelled = true;
    };
  }, [configDirty]);

  useEffect(() => {
    writeTradingPanelSessionState(state);
  }, [state.journals, state.chart, state.panelPinned]);

  useEffect(() => {
    if (configHydrationStatus !== "ready" || !configDirty) {
      return;
    }

    setConfigDirty(false);
    void writeGlobalConfig(pickStoredGlobalConfig(state));
  }, [configDirty, configHydrationStatus, state.config, state.defaultPresetSymbol, state.pairPresets]);

  const updateStoredConfig = useCallback(
    (updater: (currentState: TradingPanelState) => TradingPanelState) => {
      setState((currentState) => {
        const nextState = updater(currentState);

        if (configHydrationStatusRef.current === "ready") {
          void writeGlobalConfig(pickStoredGlobalConfig(nextState));
        } else {
          setConfigDirty(true);
        }

        return nextState;
      });
    },
    [],
  );

  useEffect(() => {
    function handleStorage(event: StorageEvent) {
      if (event.key !== TRADING_PANEL_STORAGE_KEY || event.newValue === null) {
        return;
      }

      const nextValue = event.newValue;

      try {
        setState((currentState) => ({
          ...currentState,
          ...normalizeSessionState(JSON.parse(nextValue)),
        }));
      } catch {
        setState((currentState) => ({
          ...currentState,
          ...pickSessionState(defaultState),
        }));
      }
    }

    window.addEventListener("storage", handleStorage);
    return () => {
      window.removeEventListener("storage", handleStorage);
    };
  }, []);

  const createJournal = useCallback((symbol: string, market: string) => {
    const journal = createDomainJournal({
      symbol: symbol.trim(),
      market: market.trim(),
    });

    setState((currentState) => ({
      ...currentState,
      journals: [...currentState.journals, journal],
    }));

    return journal;
  }, []);

  const finishJournal = useCallback((journalId: string, summary: string) => {
    setState((currentState) => ({
      ...currentState,
      journals: currentState.journals.map((journal) =>
        journal.id === journalId
          ? {
              ...journal,
              summary: summary.trim(),
              status: "done",
              updatedAt: new Date().toISOString(),
            }
          : journal,
      ),
    }));
  }, []);

  const recordEvent = useCallback(
    (journalId: string, draft: TradeEventDraft): TradeEventComposerResult => {
      const journal = state.journals.find((item) => item.id === journalId);
      if (!journal) {
        return { ok: false, error: "missingContent" };
      }

      const result = composeTradeEvent(
        draft,
        journal.events.reduce((total, event) => total + (event.quantity ?? 0), 0),
        journal.status,
      );

      if (!result.ok) {
        return result;
      }

      setState((currentState) => ({
        ...currentState,
        journals: currentState.journals.map((item) =>
          item.id === journalId
            ? (() => {
                const updatedAt = new Date().toISOString();

                return {
                  ...item,
                  updatedAt,
                  events: [result.event, ...item.events].sort(
                    (left, right) =>
                      new Date(right.timestamp).getTime() -
                      new Date(left.timestamp).getTime(),
                  ),
                };
              })()
            : item,
        ),
      }));

      return result;
    },
    [state.journals],
  );

  const updateConfig = useCallback((nextConfig: Partial<GlobalConfig>) => {
    updateStoredConfig((currentState) => ({
      ...currentState,
      config: {
        ...currentState.config,
        ...nextConfig,
      },
    }));
  }, [updateStoredConfig]);

  const upsertPairPreset = useCallback((symbol: string, timeframe: TimeframePreset) => {
    const normalizedSymbol = normalizeSymbol(symbol);
    if (!normalizedSymbol) {
      return;
    }

    updateStoredConfig((currentState) => {
      const nextPreset: TradingPairPreset = {
        symbol: normalizedSymbol,
        timeframe,
      };
      const hasExisting = currentState.pairPresets.some(
        (preset) => preset.symbol === normalizedSymbol,
      );

      return {
        ...currentState,
        pairPresets: hasExisting
          ? currentState.pairPresets.map((preset) =>
              preset.symbol === normalizedSymbol ? nextPreset : preset,
            )
          : [...currentState.pairPresets, nextPreset],
      };
    });
  }, [updateStoredConfig]);

  const removePairPreset = useCallback((symbol: string) => {
    const normalizedSymbol = normalizeSymbol(symbol);
    if (!normalizedSymbol) {
      return;
    }

    updateStoredConfig((currentState) => ({
      ...currentState,
      pairPresets: currentState.pairPresets.filter(
        (preset) => preset.symbol !== normalizedSymbol,
      ),
      defaultPresetSymbol:
        currentState.defaultPresetSymbol === normalizedSymbol
          ? undefined
          : currentState.defaultPresetSymbol,
    }));
  }, [updateStoredConfig]);

  const setDefaultPresetSymbol = useCallback((symbol?: string) => {
    const normalizedSymbol = typeof symbol === "string" ? normalizeSymbol(symbol) : undefined;

    updateStoredConfig((currentState) => ({
      ...currentState,
      defaultPresetSymbol:
        normalizedSymbol &&
        currentState.pairPresets.some((preset) => preset.symbol === normalizedSymbol)
          ? normalizedSymbol
          : undefined,
    }));
  }, [updateStoredConfig]);

  const updateChartView = useCallback((nextChartView: Partial<ChartViewState>) => {
    setState((currentState) => ({
      ...currentState,
      chart: normalizeChartView({
        ...currentState.chart,
        ...nextChartView,
      }),
    }));
  }, []);

  const setPanelPinned = useCallback((panelPinned: boolean) => {
    setState((currentState) => ({
      ...currentState,
      panelPinned,
    }));
  }, []);

  return useMemo(
    () => ({
      state,
      actions: {
        createJournal,
        finishJournal,
        recordEvent,
        removePairPreset,
        setDefaultPresetSymbol,
        setPanelPinned,
        upsertPairPreset,
        updateChartView,
        updateConfig,
      },
    }),
    [
      createJournal,
      finishJournal,
      recordEvent,
      removePairPreset,
      setDefaultPresetSymbol,
      setPanelPinned,
      state,
      updateChartView,
      updateConfig,
      updateStoredConfig,
      upsertPairPreset,
    ],
  );
}

function pickSessionState(state: TradingPanelState): TradingPanelSessionState {
  return {
    journals: state.journals,
    chart: state.chart,
    panelPinned: state.panelPinned,
  };
}

function normalizeSessionState(value: unknown): TradingPanelSessionState {
  return pickSessionState(normalizeState(value));
}

function pickStoredGlobalConfig(state: TradingPanelState): StoredGlobalConfig {
  return {
    config: state.config,
    pairPresets: state.pairPresets,
    defaultPresetSymbol: state.defaultPresetSymbol,
  };
}

function normalizeState(value: unknown): TradingPanelState {
  if (!isRecord(value)) {
    return defaultState;
  }

  return {
    journals: Array.isArray(value.journals)
      ? value.journals.filter(isTradeJournal)
      : defaultState.journals,
    config: isRecord(value.config)
      ? {
          timerEnabled:
            typeof value.config.timerEnabled === "boolean"
              ? value.config.timerEnabled
              : defaultState.config.timerEnabled,
          timerInterval:
            typeof value.config.timerInterval === "number"
              ? value.config.timerInterval
              : defaultState.config.timerInterval,
        }
      : defaultState.config,
    pairPresets: normalizePairPresets(value.pairPresets),
    defaultPresetSymbol: normalizeDefaultPresetSymbol(value.defaultPresetSymbol, value.pairPresets),
    chart: normalizeChartView(value.chart),
    panelPinned:
      typeof value.panelPinned === "boolean"
        ? value.panelPinned
        : defaultState.panelPinned,
  };
}

function normalizeChartView(value: unknown): ChartViewState {
  if (!isRecord(value)) {
    return defaultState.chart;
  }

  return {
    symbol:
      typeof value.symbol === "string" && CHART_SYMBOLS.has(value.symbol)
        ? value.symbol
        : defaultState.chart.symbol,
    interval:
      typeof value.interval === "string" && CHART_INTERVALS.has(value.interval)
        ? value.interval
        : defaultState.chart.interval,
  };
}

function normalizePairPresets(value: unknown): TradingPairPreset[] {
  if (!Array.isArray(value)) {
    return defaultState.pairPresets;
  }

  return value.flatMap((item) => {
    const preset = normalizePairPreset(item);
    return preset ? [preset] : [];
  });
}

function normalizePairPreset(value: unknown): TradingPairPreset | undefined {
  if (!isRecord(value) || typeof value.symbol !== "string") {
    return undefined;
  }

  const symbol = normalizeSymbol(value.symbol);
  if (!symbol || typeof value.timeframe !== "string" || !TIMEFRAME_PRESETS.has(value.timeframe)) {
    return undefined;
  }

  return {
    symbol,
    timeframe: value.timeframe as TimeframePreset,
  };
}

function normalizeDefaultPresetSymbol(
  defaultPresetSymbol: unknown,
  pairPresetsValue: unknown,
): string | undefined {
  if (typeof defaultPresetSymbol !== "string") {
    return undefined;
  }

  const normalizedSymbol = normalizeSymbol(defaultPresetSymbol);
  if (!normalizedSymbol) {
    return undefined;
  }

  const pairPresets = normalizePairPresets(pairPresetsValue);
  return pairPresets.some((preset) => preset.symbol === normalizedSymbol)
    ? normalizedSymbol
    : undefined;
}

function normalizeSymbol(value: string): string {
  return value.trim().toUpperCase();
}

function isTradeJournal(value: unknown): value is TradeJournal {
  if (!isRecord(value) || !Array.isArray(value.events)) {
    return false;
  }

  return (
    typeof value.id === "string" &&
    typeof value.createdAt === "string" &&
    typeof value.symbol === "string" &&
    typeof value.market === "string" &&
    typeof value.summary === "string" &&
    (value.status === "open" || value.status === "done")
  );
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}
