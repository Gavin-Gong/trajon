import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useTradingJournalState } from "../useTradingJournalState";

const service = {
  loadTradingPanelState: vi.fn(),
  createTradeJournal: vi.fn(),
  createTradeEvent: vi.fn(),
  finishTradeJournal: vi.fn(),
  updateChartView: vi.fn(),
  setPanelPinned: vi.fn(),
  updateConfig: vi.fn(),
  upsertPairPreset: vi.fn(),
  removePairPreset: vi.fn(),
  setDefaultPresetSymbol: vi.fn(),
};

describe("useTradingJournalState", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    service.loadTradingPanelState.mockResolvedValue({
      journals: [],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      chart: { symbol: "BINANCE:BTCUSDT", interval: "60" },
      panelPinned: true,
    });
  });

  it("hydrates from the service on mount", async () => {
    const { result } = renderHook(() => useTradingJournalState({ service }));

    await waitFor(() => {
      expect(service.loadTradingPanelState).toHaveBeenCalledTimes(1);
    });

    expect(result.current.state.chart.symbol).toBe("BINANCE:BTCUSDT");
  });

  it("creates journals through the service and updates local state", async () => {
    const createdJournal = {
      id: "journal-1",
      createdAt: "2026-04-18T00:00:00.000Z",
      updatedAt: "2026-04-18T00:00:00.000Z",
      symbol: "ETHUSDT",
      market: "BINANCE",
      summary: "",
      status: "open",
      events: [],
    };
    service.createTradeJournal.mockResolvedValue(createdJournal);

    const { result } = renderHook(() => useTradingJournalState({ service }));
    await waitFor(() => expect(service.loadTradingPanelState).toHaveBeenCalledTimes(1));

    await act(async () => {
      await result.current.actions.createJournal("ETHUSDT", "BINANCE");
    });

    expect(service.createTradeJournal).toHaveBeenCalledWith("ETHUSDT", "BINANCE");
    expect(result.current.state.journals).toEqual([createdJournal]);
  });

  it("records events through the service and updates local state", async () => {
    service.loadTradingPanelState.mockResolvedValue({
      journals: [
        {
          id: "journal-1",
          createdAt: "2026-04-18T00:00:00.000Z",
          updatedAt: "2026-04-18T00:00:00.000Z",
          symbol: "ETHUSDT",
          market: "BINANCE",
          summary: "",
          status: "open",
          events: [],
        },
      ],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      chart: { symbol: "BINANCE:BTCUSDT", interval: "60" },
      panelPinned: true,
    });
    service.createTradeEvent.mockResolvedValue({
      ok: true,
      event: {
        id: "event-1",
        createdAt: "2026-04-18T00:01:00.000Z",
        updatedAt: "2026-04-18T00:01:00.000Z",
        timestamp: "2026-04-18T00:01:00.000Z",
        type: "log",
        content: "noted",
      },
    });

    const { result } = renderHook(() => useTradingJournalState({ service }));
    await waitFor(() => expect(service.loadTradingPanelState).toHaveBeenCalledTimes(1));

    let response;
    await act(async () => {
      response = await result.current.actions.recordEvent("journal-1", {
        type: "log",
        content: "noted",
        priceInput: "",
        quantityInput: "",
      });
    });

    expect(response).toEqual(expect.objectContaining({ ok: true }));
    expect(service.createTradeEvent).toHaveBeenCalled();
    expect(result.current.state.journals[0]?.events).toHaveLength(1);
  });

  it("does not write the legacy localStorage key", async () => {
    const setItemSpy = vi.spyOn(window.localStorage.__proto__, "setItem");

    renderHook(() => useTradingJournalState({ service }));
    await waitFor(() => expect(service.loadTradingPanelState).toHaveBeenCalledTimes(1));

    expect(setItemSpy).not.toHaveBeenCalledWith(
      "trading-flow:trading-panel-state",
      expect.anything(),
    );
    setItemSpy.mockRestore();
  });
});
