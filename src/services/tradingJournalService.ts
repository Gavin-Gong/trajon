import {
  composeTradeEvent,
  createJournal,
  netPosition,
  normalizeJournalTags,
  type JournalImage,
  type TradeEventComposerResult,
  type TradeEventDraft,
  type TradeJournal,
} from "../domain/journal";
import type { TradingPanelState } from "../model/storage";
import type { GlobalConfig } from "../model/globalConfig";
import type { TimeframePreset, TradingPairPreset } from "../model/tradingPairPreset";

import { createTradingJournalRepository } from "./tradingJournalRepository";
import { appConfigValueSchema, globalConfigSchema } from "./tradingJournalSchemas";
import {
  persistJournalImages,
  removeJournalImageFile,
  type JournalImageInput,
} from "../platform/journalImages";

type TradingJournalRepository = ReturnType<typeof createTradingJournalRepository>;

export function createTradingJournalService(
  options: {
    repository?: TradingJournalRepository;
    imagePlatform?: {
      persistJournalImages: (journal: TradeJournal, files: JournalImageInput[]) => Promise<JournalImage[]>;
      removeJournalImageFile: (relativePath: string) => Promise<void>;
    };
  } = {},
) {
  const repository = options.repository ?? createTradingJournalRepository();
  const imagePlatform = options.imagePlatform ?? { persistJournalImages, removeJournalImageFile };

  return {
    loadTradingPanelState(): Promise<TradingPanelState> {
      return repository.loadState();
    },

    async createTradeJournal(symbol: string): Promise<TradeJournal> {
      const journal = createJournal({
        symbol: symbol.trim().toUpperCase(),
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

    async finishTradeJournal(journalId: string, summary: string, tags: string[] = []): Promise<void> {
      const state = await repository.loadState();
      const journal = state.journals.find((item) => item.id === journalId);
      if (!journal) {
        return;
      }

      const normalizedTags = normalizeJournalTags(tags);

      await repository.updateTradeJournal({
        ...journal,
        summary: summary.trim(),
        tags: normalizedTags,
        status: "done",
        updatedAt: new Date().toISOString(),
      });
    },

    async updateTradeJournalSymbol(journalId: string, symbol: string): Promise<void> {
      const state = await repository.loadState();
      const journal = state.journals.find((item) => item.id === journalId);
      const trimmedSymbol = symbol.trim();
      if (!journal || !trimmedSymbol) {
        return;
      }

      await repository.updateTradeJournal({
        ...journal,
        symbol: trimmedSymbol,
        updatedAt: new Date().toISOString(),
      });
    },

    async addJournalImages(journalId: string, filePaths: JournalImageInput[]): Promise<JournalImage[]> {
      const state = await repository.loadState();
      const journal = state.journals.find((item) => item.id === journalId);
      if (!journal || filePaths.length === 0) {
        return [];
      }

      const images = await imagePlatform.persistJournalImages(journal, filePaths);
      if (images.length === 0) {
        return [];
      }

      await repository.insertJournalImages(journalId, images);
      return images;
    },

    async removeJournalImage(journalId: string, imageId: string): Promise<void> {
      const state = await repository.loadState();
      const journal = state.journals.find((item) => item.id === journalId);
      const image = journal?.images.find((item) => item.id === imageId);
      if (!journal || !image) {
        return;
      }

      await repository.deleteJournalImage(imageId);
      await imagePlatform.removeJournalImageFile(image.relativePath);
    },

    async deleteTradeJournal(journalId: string): Promise<void> {
      const state = await repository.loadState();
      const journal = state.journals.find((item) => item.id === journalId);
      if (!journal) {
        return;
      }

      await Promise.all(
        journal.images.map((image) => imagePlatform.removeJournalImageFile(image.relativePath)),
      );
      await repository.deleteTradeJournal(journalId);
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
