import { useEffect, useMemo, useRef, useState } from "react";

import {
  normalizeJournalTags,
  type TradeEventComposerResult,
  type TradeEventDraft,
  type TradeJournal,
} from "../domain/journal";
import type { JournalImage } from "../domain/journal";
import type { JournalImageInput } from "../platform/journalImages";
import type { GlobalConfig } from "../model/globalConfig";
import type { TimeframePreset } from "../model/tradingPairPreset";
import type { TradingPanelState } from "../model/storage";
import {
  createTradingJournalService,
  type createTradingJournalService as createTradingJournalServiceType,
} from "../services/tradingJournalService";

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
  panelPinned: true,
};

export function useTradingJournalState(
  options: { service?: TradingJournalService } = {},
) {
  const serviceRef = useRef<TradingJournalService>(options.service ?? createTradingJournalService());
  const [state, setState] = useState<TradingPanelState>(defaultState);

  useEffect(() => {
    let cancelled = false;

    const refresh = async () => {
      const nextState = await serviceRef.current.loadTradingPanelState();
      if (!cancelled) {
        setState(nextState);
      }
    };

    void refresh();

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

    async finishJournal(journalId: string, summary: string, tags: string[] = []) {
      const normalizedTags = normalizeJournalTags(tags);
      await serviceRef.current.finishTradeJournal(journalId, summary, normalizedTags);
      setState((currentState) => ({
        ...currentState,
        journals: currentState.journals.map((journal) =>
          journal.id === journalId
            ? {
                ...journal,
                summary: summary.trim(),
                tags: normalizedTags,
                status: "done",
                updatedAt: new Date().toISOString(),
              }
            : journal,
        ),
      }));
    },

    async addJournalImages(journalId: string, filePaths: JournalImageInput[]) {
      const images = await serviceRef.current.addJournalImages(journalId, filePaths);
      setState((currentState) => ({
        ...currentState,
        journals: currentState.journals.map((journal) =>
          journal.id === journalId
            ? {
                ...journal,
                images: [...journal.images, ...images].sort((left, right) => left.sortOrder - right.sortOrder),
                updatedAt: images.length > 0 ? images[images.length - 1]?.updatedAt ?? journal.updatedAt : journal.updatedAt,
              }
            : journal,
        ),
      }));
      return images;
    },

    async removeJournalImage(journalId: string, imageId: string) {
      await serviceRef.current.removeJournalImage(journalId, imageId);
      setState((currentState) => ({
        ...currentState,
        journals: currentState.journals.map((journal) =>
          journal.id === journalId
            ? {
                ...journal,
                images: journal.images.filter((image) => image.id !== imageId),
              }
            : journal,
        ),
      }));
    },

    async deleteJournal(journalId: string) {
      await serviceRef.current.deleteTradeJournal(journalId);
      setState((currentState) => ({
        ...currentState,
        journals: currentState.journals.filter((journal) => journal.id !== journalId),
      }));
    },

    async updateJournalSymbol(journalId: string, symbol: string) {
      await serviceRef.current.updateTradeJournalSymbol(journalId, symbol);
      setState((currentState) => ({
        ...currentState,
        journals: currentState.journals.map((journal) =>
          journal.id === journalId
            ? {
                ...journal,
                symbol: symbol.trim(),
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

    async setPanelPinned(panelPinned: boolean) {
      await serviceRef.current.setPanelPinned(panelPinned);
      setState((currentState) => ({
        ...currentState,
        panelPinned,
      }));
    },

    async refreshState() {
      const nextState = await serviceRef.current.loadTradingPanelState();
      setState(nextState);
    },
  }), []);

  return {
    state,
    actions: actions as {
      createJournal: (symbol: string, market: string) => Promise<TradeJournal>;
      finishJournal: (journalId: string, summary: string, tags?: string[]) => Promise<void>;
      addJournalImages: (journalId: string, filePaths: JournalImageInput[]) => Promise<JournalImage[]>;
      removeJournalImage: (journalId: string, imageId: string) => Promise<void>;
      deleteJournal: (journalId: string) => Promise<void>;
      updateJournalSymbol: (journalId: string, symbol: string) => Promise<void>;
      recordEvent: (journalId: string, draft: TradeEventDraft) => Promise<TradeEventComposerResult>;
      updateConfig: (nextConfig: Partial<GlobalConfig>) => Promise<void>;
      upsertPairPreset: (symbol: string, timeframe: TimeframePreset) => Promise<void>;
      removePairPreset: (symbol: string) => Promise<void>;
      setDefaultPresetSymbol: (symbol?: string) => Promise<void>;
      setPanelPinned: (panelPinned: boolean) => Promise<void>;
      refreshState: () => Promise<void>;
    },
  };
}
