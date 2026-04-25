import { useEffect, useMemo, useSyncExternalStore } from "react";

import type {
  TradeEventComposerResult,
  TradeEventDraft,
  TradeJournal,
} from "../domain/journal";
import type { JournalImage } from "../domain/journal";
import type { GlobalConfig } from "../model/globalConfig";
import type { TimeframePreset } from "../model/tradingPairPreset";
import type { TradingPanelState } from "../model/storage";
import type { JournalImageInput } from "../platform/journalImages";

import { getTradingJournalStateStore, type TradingJournalService } from "./tradingJournalStateStore";

export type { GlobalConfig, TradingPanelState } from "../model/storage";

export function useTradingJournalState(
  options: { service?: TradingJournalService } = {},
) {
  const store = useMemo(() => getTradingJournalStateStore(options.service), [options.service]);
  const state = useSyncExternalStore(store.subscribe, store.getSnapshot, store.getSnapshot);

  useEffect(() => {
    void store.initialize();
  }, [store]);

  return {
    state,
    actions: store.actions as {
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
      refreshState: () => Promise<TradingPanelState>;
    },
  };
}
