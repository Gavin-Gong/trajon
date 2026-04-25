import type { TradeEventComposerResult, TradeEventDraft } from "../domain/journal";
import { normalizeJournalTags } from "../domain/journal";
import type { GlobalConfig } from "../model/globalConfig";
import type { TimeframePreset } from "../model/tradingPairPreset";
import type { TradingPanelState } from "../model/storage";
import type { JournalImageInput } from "../platform/journalImages";
import {
  createTradingJournalService,
  type createTradingJournalService as createTradingJournalServiceType,
} from "../services/tradingJournalService";

type TradingJournalService = ReturnType<typeof createTradingJournalServiceType>;
type Listener = () => void;

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

const TRADING_JOURNAL_SYNC_CHANNEL = "trading-journal-state";

type TradingJournalStateStore = ReturnType<typeof createTradingJournalStateStore>;

const defaultService = createTradingJournalService();
const storeCache = new WeakMap<TradingJournalService, TradingJournalStateStore>();

function cloneState(state: TradingPanelState): TradingPanelState {
  return {
    ...state,
    journals: state.journals.map((journal) => ({
      ...journal,
      tags: [...(journal.tags ?? [])],
      events: [...(journal.events ?? [])],
      images: [...(journal.images ?? [])],
    })),
    config: { ...state.config },
    pairPresets: state.pairPresets.map((preset) => ({ ...preset })),
  };
}

function createCrossWindowSync(onExternalChange: () => void) {
  if (typeof window === "undefined" || typeof BroadcastChannel === "undefined") {
    return {
      notify() {
        return;
      },
      dispose() {
        return;
      },
    };
  }

  const channel = new BroadcastChannel(TRADING_JOURNAL_SYNC_CHANNEL);
  const handleMessage = () => {
    onExternalChange();
  };

  channel.addEventListener("message", handleMessage);

  return {
    notify() {
      channel.postMessage({ type: "state-changed", timestamp: Date.now() });
    },
    dispose() {
      channel.removeEventListener("message", handleMessage);
      channel.close();
    },
  };
}

function createTradingJournalStateStore(service: TradingJournalService) {
  let state = defaultState;
  let initialized = false;
  let refreshPromise: Promise<TradingPanelState> | undefined;
  const listeners = new Set<Listener>();

  const sync = createCrossWindowSync(() => {
    void store.refresh();
  });

  function emit() {
    listeners.forEach((listener) => listener());
  }

  function setSnapshot(nextState: TradingPanelState) {
    state = nextState;
    emit();
  }

  async function mutate<T>(run: () => Promise<T>, updateSnapshot?: (currentState: TradingPanelState, result: T) => TradingPanelState) {
    const result = await run();

    if (updateSnapshot) {
      setSnapshot(updateSnapshot(state, result));
    } else {
      await store.refresh();
    }

    sync.notify();
    return result;
  }

  const store = {
    getSnapshot() {
      return state;
    },

    subscribe(listener: Listener) {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },

    async initialize() {
      if (initialized) {
        return state;
      }

      initialized = true;
      return store.refresh();
    },

    async refresh() {
      if (!refreshPromise) {
        refreshPromise = service.loadTradingPanelState().then((nextState) => {
          setSnapshot(cloneState(nextState));
          return state;
        }).finally(() => {
          refreshPromise = undefined;
        });
      }

      return refreshPromise;
    },

    actions: {
      createJournal(symbol: string, market: string) {
        return mutate(
          () => service.createTradeJournal(symbol, market),
          (currentState, journal) => ({
            ...currentState,
            journals: [...currentState.journals, journal],
          }),
        );
      },

      finishJournal(journalId: string, summary: string, tags: string[] = []) {
        const normalizedTags = normalizeJournalTags(tags);
        return mutate(
          () => service.finishTradeJournal(journalId, summary, normalizedTags),
          (currentState) => ({
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
          }),
        );
      },

      addJournalImages(journalId: string, filePaths: JournalImageInput[]) {
        return mutate(
          () => service.addJournalImages(journalId, filePaths),
          (currentState, images) => ({
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
          }),
        );
      },

      removeJournalImage(journalId: string, imageId: string) {
        return mutate(
          () => service.removeJournalImage(journalId, imageId),
          (currentState) => ({
            ...currentState,
            journals: currentState.journals.map((journal) =>
              journal.id === journalId
                ? {
                    ...journal,
                    images: journal.images.filter((image) => image.id !== imageId),
                  }
                : journal,
            ),
          }),
        );
      },

      deleteJournal(journalId: string) {
        return mutate(
          () => service.deleteTradeJournal(journalId),
          (currentState) => ({
            ...currentState,
            journals: currentState.journals.filter((journal) => journal.id !== journalId),
          }),
        );
      },

      updateJournalSymbol(journalId: string, symbol: string) {
        return mutate(
          () => service.updateTradeJournalSymbol(journalId, symbol),
          (currentState) => ({
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
          }),
        );
      },

      async recordEvent(journalId: string, draft: TradeEventDraft): Promise<TradeEventComposerResult> {
        return mutate(
          () => service.createTradeEvent(journalId, draft),
          (currentState, result) => {
            if (!result.ok) {
              return currentState;
            }

            return {
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
            };
          },
        );
      },

      updateConfig(nextConfig: Partial<GlobalConfig>) {
        return mutate(
          () => service.updateConfig(nextConfig),
          (currentState) => ({
            ...currentState,
            config: {
              ...currentState.config,
              ...nextConfig,
            },
          }),
        );
      },

      upsertPairPreset(symbol: string, timeframe: TimeframePreset) {
        return mutate(
          () => service.upsertPairPreset(symbol, timeframe),
          (currentState) => {
            const normalizedSymbol = symbol.trim().toUpperCase();
            if (!normalizedSymbol) {
              return currentState;
            }

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
          },
        );
      },

      removePairPreset(symbol: string) {
        return mutate(
          () => service.removePairPreset(symbol),
          (currentState) => {
            const normalizedSymbol = symbol.trim().toUpperCase();
            return {
              ...currentState,
              pairPresets: currentState.pairPresets.filter((preset) => preset.symbol !== normalizedSymbol),
              defaultPresetSymbol:
                currentState.defaultPresetSymbol === normalizedSymbol
                  ? undefined
                  : currentState.defaultPresetSymbol,
            };
          },
        );
      },

      setDefaultPresetSymbol(symbol?: string) {
        return mutate(
          () => service.setDefaultPresetSymbol(symbol),
          (currentState) => {
            const normalizedSymbol = symbol?.trim().toUpperCase();
            return {
              ...currentState,
              defaultPresetSymbol:
                normalizedSymbol && currentState.pairPresets.some((preset) => preset.symbol === normalizedSymbol)
                  ? normalizedSymbol
                  : undefined,
            };
          },
        );
      },

      setPanelPinned(panelPinned: boolean) {
        return mutate(
          () => service.setPanelPinned(panelPinned),
          (currentState) => ({
            ...currentState,
            panelPinned,
          }),
        );
      },

      refreshState() {
        return store.refresh();
      },
    },
  };

  return store;
}

export function getTradingJournalStateStore(service?: TradingJournalService) {
  const resolvedService = service ?? defaultService;
  const cached = storeCache.get(resolvedService);
  if (cached) {
    return cached;
  }

  const store = createTradingJournalStateStore(resolvedService);
  storeCache.set(resolvedService, store);
  return store;
}

export function resetTradingJournalStateStore(service?: TradingJournalService) {
  const resolvedService = service ?? defaultService;
  const store = storeCache.get(resolvedService);
  if (store) {
    storeCache.delete(resolvedService);
  }
}

export type { TradingJournalService };
