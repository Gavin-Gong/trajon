import { useCallback, useEffect, useMemo, useState } from "react";

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
import { TRADING_PANEL_STORAGE_KEY, type TradingPanelState } from "./model/storage";

export type GlobalConfig = {
  timerEnabled: boolean;
  timerInterval: number;
};

export { TRADING_PANEL_STORAGE_KEY, type TradingPanelState } from "./model/storage";

const defaultState: TradingPanelState = {
  journals: [],
  config: {
    timerEnabled: false,
    timerInterval: 300,
  },
  pairPresets: [],
  defaultPresetSymbol: undefined,
  panelPinned: true,
};

const TIMEFRAME_PRESETS = new Set<string>(TIMEFRAME_PRESET_VALUES);

export function readTradingPanelState(): TradingPanelState {
  if (typeof window === "undefined") {
    return defaultState;
  }

  const raw = window.localStorage.getItem(TRADING_PANEL_STORAGE_KEY);
  if (!raw) {
    return defaultState;
  }

  try {
    return normalizeState(JSON.parse(raw));
  } catch {
    return defaultState;
  }
}

function writeTradingPanelState(state: TradingPanelState) {
  if (typeof window === "undefined") {
    return;
  }

  window.localStorage.setItem(TRADING_PANEL_STORAGE_KEY, JSON.stringify(state));
}

export function useTradingPanelState() {
  const [state, setState] = useState<TradingPanelState>(() => readTradingPanelState());

  useEffect(() => {
    writeTradingPanelState(state);
  }, [state]);

  useEffect(() => {
    function handleStorage(event: StorageEvent) {
      if (event.key !== TRADING_PANEL_STORAGE_KEY || event.newValue === null) {
        return;
      }

      try {
        setState(normalizeState(JSON.parse(event.newValue)));
      } catch {
        setState(defaultState);
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
            ? {
                ...item,
                events: [result.event, ...item.events].sort(
                  (left, right) =>
                    new Date(right.timestamp).getTime() -
                    new Date(left.timestamp).getTime(),
                ),
              }
            : item,
        ),
      }));

      return result;
    },
    [state.journals],
  );

  const updateConfig = useCallback((nextConfig: Partial<GlobalConfig>) => {
    setState((currentState) => ({
      ...currentState,
      config: {
        ...currentState.config,
        ...nextConfig,
      },
    }));
  }, []);

  const upsertPairPreset = useCallback((symbol: string, timeframe: TimeframePreset) => {
    const normalizedSymbol = normalizeSymbol(symbol);
    if (!normalizedSymbol) {
      return;
    }

    setState((currentState) => {
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
  }, []);

  const removePairPreset = useCallback((symbol: string) => {
    const normalizedSymbol = normalizeSymbol(symbol);
    if (!normalizedSymbol) {
      return;
    }

    setState((currentState) => ({
      ...currentState,
      pairPresets: currentState.pairPresets.filter(
        (preset) => preset.symbol !== normalizedSymbol,
      ),
      defaultPresetSymbol:
        currentState.defaultPresetSymbol === normalizedSymbol
          ? undefined
          : currentState.defaultPresetSymbol,
    }));
  }, []);

  const setDefaultPresetSymbol = useCallback((symbol?: string) => {
    const normalizedSymbol = typeof symbol === "string" ? normalizeSymbol(symbol) : undefined;

    setState((currentState) => ({
      ...currentState,
      defaultPresetSymbol:
        normalizedSymbol &&
        currentState.pairPresets.some((preset) => preset.symbol === normalizedSymbol)
          ? normalizedSymbol
          : undefined,
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
      updateConfig,
      upsertPairPreset,
    ],
  );
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
    panelPinned:
      typeof value.panelPinned === "boolean"
        ? value.panelPinned
        : defaultState.panelPinned,
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
