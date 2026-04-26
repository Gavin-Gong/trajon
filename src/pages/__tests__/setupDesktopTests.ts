import { vi } from "vitest";

import type { TradingPanelState } from "../../model/storage";

const mockedStoreState = vi.hoisted(() => ({
  value: undefined as TradingPanelState | undefined,
}));

const mockedRefreshState = vi.hoisted(() => vi.fn(() => Promise.resolve()));

vi.mock("../../stores/useTradingJournalState", async () => {
  const React = await import("react");
  const domain = await import("../../domain/journal");

  function cloneState(state: TradingPanelState): TradingPanelState {
    return JSON.parse(JSON.stringify(state)) as TradingPanelState;
  }

  function buildState(): TradingPanelState {
    const openJournal = domain.createJournal({
      id: "open-journal",
      symbol: "NQ",
      createdAt: "2026-04-04T09:30:00.000Z",
      status: "open",
      summary: "等待二次确认后继续执行。",
    });
    const doneJournal = domain.createJournal({
      id: "done-journal",
      symbol: "ES",
      createdAt: "2026-04-02T09:30:00.000Z",
      status: "done",
      summary: "纪律不错，但出场可以更果断。",
    });

    const entry = domain.composeTradeEvent(
      {
        type: "entry",
        content: "Opening drive long",
        quantityInput: "2",
        priceInput: "18250",
      },
      0,
      "open",
    );
    const log = domain.composeTradeEvent(
      {
        type: "log",
        content: "保持耐心，等待回踩确认。",
        quantityInput: "",
        priceInput: "",
      },
      2,
      "open",
    );

    if (entry.ok && log.ok) {
      openJournal.events = [entry.event, log.event];
    }

    return {
      journals: [doneJournal, openJournal],
      config: {
        timerEnabled: true,
        timerInterval: 300,
      },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      panelPinned: true,
    };
  }

  const listeners = new Set<() => void>();

  function ensureState(): TradingPanelState {
    if (!mockedStoreState.value) {
      mockedStoreState.value = cloneState(buildState());
    }

    return mockedStoreState.value;
  }

  function readState(): TradingPanelState {
    return ensureState();
  }

  function writeState(nextState: TradingPanelState) {
    mockedStoreState.value = cloneState(nextState);
    listeners.forEach((listener) => listener());
  }

  function updateState(updater: (currentState: TradingPanelState) => TradingPanelState) {
    writeState(updater(readState()));
  }

  return {
    useTradingJournalState() {
      const state = React.useSyncExternalStore(
        (listener) => {
          listeners.add(listener);
          return () => listeners.delete(listener);
        },
        readState,
        readState,
      );

      return {
        state,
        actions: {
          createJournal(symbol: string) {
            const journal = domain.createJournal({ symbol: symbol.trim() });
            updateState((currentState) => ({
              ...currentState,
              journals: [...currentState.journals, journal],
            }));
            return Promise.resolve(journal);
          },
          finishJournal(journalId: string, summary: string, tags: string[] = []) {
            updateState((currentState) => ({
              ...currentState,
              journals: currentState.journals.map((journal) =>
                journal.id === journalId
                  ? {
                      ...journal,
                      summary: summary.trim(),
                      tags,
                      status: "done",
                      updatedAt: new Date().toISOString(),
                    }
                  : journal,
              ),
            }));
            return Promise.resolve();
          },
          recordEvent(journalId: string, draft: Parameters<typeof domain.composeTradeEvent>[0]) {
            const journal = readState().journals.find((item) => item.id === journalId);
            if (!journal) {
              return Promise.resolve({ ok: false, error: "missingContent" } as const);
            }

            const result = domain.composeTradeEvent(
              draft,
              journal.events.reduce((total, event) => total + (event.quantity ?? 0), 0),
              journal.status,
            );
            if (!result.ok) {
              return Promise.resolve(result);
            }

            updateState((currentState) => ({
              ...currentState,
              journals: currentState.journals.map((item) =>
                item.id === journalId
                  ? {
                      ...item,
                      updatedAt: result.event.updatedAt,
                      events: [result.event, ...item.events],
                    }
                  : item,
                ),
            }));
            return Promise.resolve(result);
          },
          addJournalImages(journalId: string, files: File[]) {
            updateState((currentState) => ({
              ...currentState,
              journals: currentState.journals.map((journal) =>
                journal.id === journalId
                  ? {
                      ...journal,
                      images: [
                        ...journal.images,
                        ...files.map((file, index) => ({
                          id: `${journalId}-image-${journal.images.length + index + 1}`,
                          journalId,
                          createdAt: new Date().toISOString(),
                          updatedAt: new Date().toISOString(),
                          fileName: file.name,
                          relativePath: `trajon/images/${file.name}`,
                          sortOrder: journal.images.length + index,
                        })),
                      ],
                    }
                  : journal,
              ),
            }));
            return Promise.resolve([]);
          },
          removeJournalImage(journalId: string, imageId: string) {
            updateState((currentState) => ({
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
            return Promise.resolve();
          },
          deleteJournal(journalId: string) {
            updateState((currentState) => ({
              ...currentState,
              journals: currentState.journals.filter((journal) => journal.id !== journalId),
            }));
            return Promise.resolve();
          },
          updateJournalSymbol(journalId: string, symbol: string) {
            updateState((currentState) => ({
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
            return Promise.resolve();
          },
          updateConfig(nextConfig: Partial<TradingPanelState["config"]>) {
            updateState((currentState) => ({
              ...currentState,
              config: { ...currentState.config, ...nextConfig },
            }));
            return Promise.resolve();
          },
          upsertPairPreset(symbol: string, timeframe: TradingPanelState["pairPresets"][number]["timeframe"]) {
            const normalizedSymbol = symbol.trim().toUpperCase();
            if (!normalizedSymbol) {
              return Promise.resolve();
            }

            updateState((currentState) => {
              const nextPreset = { symbol: normalizedSymbol, timeframe };
              const hasExisting = currentState.pairPresets.some((preset) => preset.symbol === normalizedSymbol);
              return {
                ...currentState,
                pairPresets: hasExisting
                  ? currentState.pairPresets.map((preset) =>
                      preset.symbol === normalizedSymbol ? nextPreset : preset,
                    )
                  : [...currentState.pairPresets, nextPreset],
              };
            });
            return Promise.resolve();
          },
          removePairPreset(symbol: string) {
            const normalizedSymbol = symbol.trim().toUpperCase();
            updateState((currentState) => ({
              ...currentState,
              pairPresets: currentState.pairPresets.filter((preset) => preset.symbol !== normalizedSymbol),
              defaultPresetSymbol:
                currentState.defaultPresetSymbol === normalizedSymbol
                  ? undefined
                  : currentState.defaultPresetSymbol,
            }));
            return Promise.resolve();
          },
          setDefaultPresetSymbol(symbol?: string) {
            const normalizedSymbol = symbol?.trim().toUpperCase();
            updateState((currentState) => ({
              ...currentState,
              defaultPresetSymbol:
                normalizedSymbol && currentState.pairPresets.some((preset) => preset.symbol === normalizedSymbol)
                  ? normalizedSymbol
                  : undefined,
            }));
            return Promise.resolve();
          },
          setPanelPinned(panelPinned: boolean) {
            updateState((currentState) => ({
              ...currentState,
              panelPinned,
            }));
            return Promise.resolve();
          },
          refreshState() {
            listeners.forEach((listener) => listener());
            return mockedRefreshState();
          },
        },
      };
    },
  };
});

vi.mock("../../platform/panelWindow", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../platform/panelWindow")>();
  return {
    ...actual,
    openPanelWindow: vi.fn(() => Promise.resolve()),
  };
});

vi.mock("../../platform/panelLifecycle", () => ({
  closePanelWindow: vi.fn(() => Promise.resolve()),
}));

export function getMockedStoreState() {
  return mockedStoreState;
}

export function getMockedRefreshState() {
  return mockedRefreshState;
}
