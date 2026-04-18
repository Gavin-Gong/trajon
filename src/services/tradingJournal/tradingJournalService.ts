import {
  composeTradeEvent,
  createJournal,
  netPosition,
  type TradeEventComposerResult,
  type TradeEventDraft,
  type TradeJournal,
} from "../../domain/tradingJournalDomain";
import type { TradingPanelState } from "../../model/storage";
import type { GlobalConfig } from "../../model/globalConfig";
import type { TimeframePreset, TradingPairPreset } from "../../model/tradingPairPreset";

import { createTradingJournalRepository } from "./tradingJournalRepository";
import { appConfigValueSchema, globalConfigSchema } from "./tradingJournalSchemas";

type TradingJournalRepository = ReturnType<typeof createTradingJournalRepository>;

export function createTradingJournalService(
  options: { repository?: TradingJournalRepository } = {},
) {
  const repository = options.repository ?? createTradingJournalRepository();

  return {
    loadTradingPanelState(): Promise<TradingPanelState> {
      return repository.loadState();
    },

    async createTradeJournal(symbol: string, market: string): Promise<TradeJournal> {
      const journal = createJournal({
        symbol: symbol.trim().toUpperCase(),
        market: market.trim(),
      });
      await repository.insertTradeJournal(journal);
      return journal;
    },

    async createTradeEvent(
      journalId: string,
      draft: TradeEventDraft,
    ): Promise<TradeEventComposerResult> {
      const state = await repository.loadState();
      const journal = state.journals.find((item) => item.id === journalId);
      if (!journal) {
        return { ok: false, error: "missingContent" };
      }

      const result = composeTradeEvent(draft, netPosition(journal.events), journal.status);
      if (!result.ok) {
        return result;
      }

      await repository.appendTradeEvent(journalId, result.event);
      return result;
    },

    async finishTradeJournal(journalId: string, summary: string): Promise<void> {
      const state = await repository.loadState();
      const journal = state.journals.find((item) => item.id === journalId);
      if (!journal) {
        return;
      }

      await repository.updateTradeJournal({
        ...journal,
        summary: summary.trim(),
        status: "done",
        updatedAt: new Date().toISOString(),
      });
    },

    async setPanelPinned(panelPinned: boolean): Promise<void> {
      await repository.writeAppState({ panelPinned });
    },

    async updateConfig(nextConfig: Partial<GlobalConfig>): Promise<void> {
      const state = await repository.loadState();
      await repository.writeAppState({
        config: globalConfigSchema.parse({
          ...state.config,
          ...nextConfig,
        }),
      });
    },

    async upsertPairPreset(symbol: string, timeframe: TimeframePreset): Promise<void> {
      const state = await repository.loadState();
      const normalizedSymbol = symbol.trim().toUpperCase();
      if (!normalizedSymbol) {
        return;
      }

      const nextPreset: TradingPairPreset = { symbol: normalizedSymbol, timeframe };
      const pairPresets = state.pairPresets.some((preset) => preset.symbol === normalizedSymbol)
        ? state.pairPresets.map((preset) => (preset.symbol === normalizedSymbol ? nextPreset : preset))
        : [...state.pairPresets, nextPreset];

      await repository.writeAppState({ pairPresets });
    },

    async removePairPreset(symbol: string): Promise<void> {
      const state = await repository.loadState();
      const normalizedSymbol = symbol.trim().toUpperCase();
      if (!normalizedSymbol) {
        return;
      }

      await repository.writeAppState({
        pairPresets: state.pairPresets.filter((preset) => preset.symbol !== normalizedSymbol),
        defaultPresetSymbol:
          state.defaultPresetSymbol === normalizedSymbol ? undefined : state.defaultPresetSymbol,
      });
    },

    async setDefaultPresetSymbol(symbol?: string): Promise<void> {
      const state = await repository.loadState();
      const normalizedSymbol = symbol?.trim().toUpperCase();
      const nextValue = normalizedSymbol && state.pairPresets.some((preset) => preset.symbol === normalizedSymbol)
        ? normalizedSymbol
        : undefined;

      await repository.writeAppState(appConfigValueSchema.parse({ defaultPresetSymbol: nextValue }));
    },
  };
}
