import { useEffect, useMemo, useRef, useState } from "react";

import type { TradeEventComposerResult, TradeEventDraft, TradeJournal } from "../domain/tradingJournalDomain";
import type { GlobalConfig } from "../model/globalConfig";
import type { TimeframePreset } from "../model/tradingPairPreset";
import type { ChartViewState, TradingPanelState } from "../model/storage";
import {
  createTradingJournalService,
  type createTradingJournalService as createTradingJournalServiceType,
} from "../services/tradingJournal/tradingJournalService";

export type { GlobalConfig, TradingPanelState } from "../model/storage";

type TradingJournalService = ReturnType<typeof createTradingJournalServiceType>;

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

export function useTradingJournalState(
  options: { service?: TradingJournalService } = {},
) {
  const serviceRef = useRef<TradingJournalService>(options.service ?? createTradingJournalService());
  const [state, setState] = useState<TradingPanelState>(defaultState);

  useEffect(() => {
    let cancelled = false;

    void serviceRef.current.loadTradingPanelState().then((nextState) => {
      if (!cancelled) {
        setState(nextState);
      }
    });

    return () => {
      cancelled = true;
    };
  }, []);

  const actions = useMemo(() => ({
    async createJournal(symbol: string, market: string) {
      const journal = await serviceRef.current.createTradeJournal(symbol, market);
      setState((currentState) => ({
        ...currentState,
        journals: [...currentState.journals, journal],
      }));
      return journal;
    },

    async finishJournal(journalId: string, summary: string) {
      await serviceRef.current.finishTradeJournal(journalId, summary);
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
    },

    async recordEvent(
      journalId: string,
      draft: TradeEventDraft,
    ): Promise<TradeEventComposerResult> {
      const result = await serviceRef.current.createTradeEvent(journalId, draft);
      if (!result.ok) {
        return result;
      }

      setState((currentState) => ({
        ...currentState,
        journals: currentState.journals.map((journal) =>
          journal.id === journalId
            ? {
                ...journal,
                updatedAt: result.event.updatedAt,
                events: [result.event, ...journal.events],
              }
            : journal,
        ),
      }));

      return result;
    },

    async updateConfig(nextConfig: Partial<GlobalConfig>) {
      await serviceRef.current.updateConfig(nextConfig);
      setState((currentState) => ({
        ...currentState,
        config: {
          ...currentState.config,
          ...nextConfig,
        },
      }));
    },

    async upsertPairPreset(symbol: string, timeframe: TimeframePreset) {
      await serviceRef.current.upsertPairPreset(symbol, timeframe);
      const normalizedSymbol = symbol.trim().toUpperCase();
      if (!normalizedSymbol) {
        return;
      }

      setState((currentState) => {
        const nextPreset = { symbol: normalizedSymbol, timeframe };
        const exists = currentState.pairPresets.some((preset) => preset.symbol === normalizedSymbol);
        return {
          ...currentState,
          pairPresets: exists
            ? currentState.pairPresets.map((preset) =>
                preset.symbol === normalizedSymbol ? nextPreset : preset,
              )
            : [...currentState.pairPresets, nextPreset],
        };
      });
    },

    async removePairPreset(symbol: string) {
      await serviceRef.current.removePairPreset(symbol);
      const normalizedSymbol = symbol.trim().toUpperCase();
      setState((currentState) => ({
        ...currentState,
        pairPresets: currentState.pairPresets.filter((preset) => preset.symbol !== normalizedSymbol),
        defaultPresetSymbol:
          currentState.defaultPresetSymbol === normalizedSymbol
            ? undefined
            : currentState.defaultPresetSymbol,
      }));
    },

    async setDefaultPresetSymbol(symbol?: string) {
      await serviceRef.current.setDefaultPresetSymbol(symbol);
      const normalizedSymbol = symbol?.trim().toUpperCase();
      setState((currentState) => ({
        ...currentState,
        defaultPresetSymbol:
          normalizedSymbol && currentState.pairPresets.some((preset) => preset.symbol === normalizedSymbol)
            ? normalizedSymbol
            : undefined,
      }));
    },

    async updateChartView(nextChartView: Partial<ChartViewState>) {
      await serviceRef.current.updateChartView(nextChartView);
      setState((currentState) => ({
        ...currentState,
        chart: {
          ...currentState.chart,
          ...nextChartView,
        },
      }));
    },

    async setPanelPinned(panelPinned: boolean) {
      await serviceRef.current.setPanelPinned(panelPinned);
      setState((currentState) => ({
        ...currentState,
        panelPinned,
      }));
    },
  }), []);

  return {
    state,
    actions: actions as {
      createJournal: (symbol: string, market: string) => Promise<TradeJournal>;
      finishJournal: (journalId: string, summary: string) => Promise<void>;
      recordEvent: (journalId: string, draft: TradeEventDraft) => Promise<TradeEventComposerResult>;
      updateConfig: (nextConfig: Partial<GlobalConfig>) => Promise<void>;
      upsertPairPreset: (symbol: string, timeframe: TimeframePreset) => Promise<void>;
      removePairPreset: (symbol: string) => Promise<void>;
      setDefaultPresetSymbol: (symbol?: string) => Promise<void>;
      updateChartView: (nextChartView: Partial<ChartViewState>) => Promise<void>;
      setPanelPinned: (panelPinned: boolean) => Promise<void>;
    },
  };
}
