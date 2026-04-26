import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { resetTradingJournalStateStore } from "../tradingJournalStateStore";
import { useTradingJournalState } from "../useTradingJournalState";

const service = {
  loadTradingPanelState: vi.fn(),
  createTradeJournal: vi.fn(),
  createTradeEvent: vi.fn(),
  finishTradeJournal: vi.fn(),
  updateTradeJournalSymbol: vi.fn(),
  addJournalImages: vi.fn(),
  removeJournalImage: vi.fn(),
  deleteTradeJournal: vi.fn(),
  setPanelPinned: vi.fn(),
  updateConfig: vi.fn(),
  upsertPairPreset: vi.fn(),
  removePairPreset: vi.fn(),
  setDefaultPresetSymbol: vi.fn(),
};

describe("useTradingJournalState", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetTradingJournalStateStore(service as never);
    service.loadTradingPanelState.mockResolvedValue({
      journals: [],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      panelPinned: true,
    });
  });

  it("adds images through the service and updates local state", async () => {
    service.loadTradingPanelState.mockResolvedValue({
      journals: [
        {
          id: "journal-1",
          createdAt: "2026-04-18T00:00:00.000Z",
          updatedAt: "2026-04-18T00:00:00.000Z",
          symbol: "ETHUSDT",
          summary: "",
          status: "open",
          events: [],
          images: [],
        },
      ],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      panelPinned: true,
    });
    service.addJournalImages.mockResolvedValue([
      {
        id: "image-1",
        journalId: "journal-1",
        createdAt: "2026-04-18T00:01:00.000Z",
        updatedAt: "2026-04-18T00:01:00.000Z",
        fileName: "journal-1-1713398460000-ethusdt.png",
        relativePath: "images/journal-1-1713398460000-ethusdt.png",
        sortOrder: 0,
      },
    ]);

    const { result } = renderHook(() => useTradingJournalState({ service }));
    await waitFor(() => expect(service.loadTradingPanelState).toHaveBeenCalledTimes(1));

    await act(async () => {
      await result.current.actions.addJournalImages("journal-1", ["/tmp/chart.png"]);
    });

    expect(service.addJournalImages).toHaveBeenCalledWith("journal-1", ["/tmp/chart.png"]);
    expect(result.current.state.journals[0]?.images).toHaveLength(1);
  });

  it("removes an image through the service and updates local state", async () => {
    service.loadTradingPanelState.mockResolvedValue({
      journals: [
        {
          id: "journal-1",
          createdAt: "2026-04-18T00:00:00.000Z",
          updatedAt: "2026-04-18T00:00:00.000Z",
          symbol: "ETHUSDT",
          summary: "",
          status: "open",
          events: [],
          images: [
            {
              id: "image-1",
              journalId: "journal-1",
              createdAt: "2026-04-18T00:01:00.000Z",
              updatedAt: "2026-04-18T00:01:00.000Z",
              fileName: "journal-1-1713398460000-ethusdt.png",
              relativePath: "images/journal-1-1713398460000-ethusdt.png",
              sortOrder: 0,
            },
          ],
        },
      ],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      panelPinned: true,
    });
    service.removeJournalImage.mockResolvedValue(undefined);

    const { result } = renderHook(() => useTradingJournalState({ service }));
    await waitFor(() => expect(service.loadTradingPanelState).toHaveBeenCalledTimes(1));

    await act(async () => {
      await result.current.actions.removeJournalImage("journal-1", "image-1");
    });

    expect(service.removeJournalImage).toHaveBeenCalledWith("journal-1", "image-1");
    expect(result.current.state.journals[0]?.images).toEqual([]);
  });

  it("hydrates from the service on mount", async () => {
    const { result } = renderHook(() => useTradingJournalState({ service }));

    await waitFor(() => {
      expect(service.loadTradingPanelState).toHaveBeenCalledTimes(1);
    });

    expect(result.current.state.panelPinned).toBe(true);
  });

  it("creates journals through the service and updates local state", async () => {
    const createdJournal = {
      id: "journal-1",
      createdAt: "2026-04-18T00:00:00.000Z",
      updatedAt: "2026-04-18T00:00:00.000Z",
      symbol: "ETHUSDT",
      summary: "",
      status: "open",
      events: [],
      images: [],
      tags: [],
    };
    service.createTradeJournal.mockResolvedValue(createdJournal);

    const { result } = renderHook(() => useTradingJournalState({ service }));
    await waitFor(() => expect(service.loadTradingPanelState).toHaveBeenCalledTimes(1));

    await act(async () => {
      await result.current.actions.createJournal("ETHUSDT");
    });

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
          summary: "",
          status: "open",
          events: [],
          images: [],
        },
      ],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
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
      "trajon:trading-panel-state",
      expect.anything(),
    );
    setItemSpy.mockRestore();
  });

  it("updates a journal symbol through the service and local state", async () => {
    service.loadTradingPanelState.mockResolvedValue({
      journals: [
        {
          id: "journal-1",
          createdAt: "2026-04-18T00:00:00.000Z",
          updatedAt: "2026-04-18T00:00:00.000Z",
          symbol: "",
          summary: "",
          status: "open",
          events: [],
          images: [],
          tags: [],
        },
      ],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      panelPinned: true,
    });
    service.updateTradeJournalSymbol.mockResolvedValue(undefined);

    const { result } = renderHook(() => useTradingJournalState({ service }));
    await waitFor(() => expect(service.loadTradingPanelState).toHaveBeenCalledTimes(1));

    await act(async () => {
      await result.current.actions.updateJournalSymbol("journal-1", "BTCUSDT");
    });

    expect(service.updateTradeJournalSymbol).toHaveBeenCalledWith("journal-1", "BTCUSDT");
    expect(result.current.state.journals[0]).toEqual(
      expect.objectContaining({
        symbol: "BTCUSDT",
      }),
    );
  });

  it("finishes journals with tags and updates local state", async () => {
    service.loadTradingPanelState.mockResolvedValue({
      journals: [
        {
          id: "journal-1",
          createdAt: "2026-04-18T00:00:00.000Z",
          updatedAt: "2026-04-18T00:00:00.000Z",
          symbol: "ETHUSDT",
          summary: "",
          status: "open",
          events: [],
          images: [],
          tags: [],
        },
      ],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      panelPinned: true,
    });
    service.finishTradeJournal.mockResolvedValue(undefined);

    const { result } = renderHook(() => useTradingJournalState({ service }));
    await waitFor(() => expect(service.loadTradingPanelState).toHaveBeenCalledTimes(1));

    await act(async () => {
      await result.current.actions.finishJournal("journal-1", "  wrapped up  ", ["breakout", "FOMO"]);
    });

    expect(service.finishTradeJournal).toHaveBeenCalledWith("journal-1", "  wrapped up  ", ["breakout", "FOMO"]);
    expect(result.current.state.journals[0]).toEqual(
      expect.objectContaining({
        summary: "wrapped up",
        status: "done",
        tags: ["breakout", "FOMO"],
      }),
    );
  });

  it("shares finished journal updates across hook instances", async () => {
    service.loadTradingPanelState.mockResolvedValue({
      journals: [
        {
          id: "journal-1",
          createdAt: "2026-04-18T00:00:00.000Z",
          updatedAt: "2026-04-18T00:00:00.000Z",
          symbol: "ETHUSDT",
          summary: "",
          status: "open",
          events: [],
          images: [],
          tags: [],
        },
      ],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      panelPinned: true,
    });

    const left = renderHook(() => useTradingJournalState({ service }));
    const right = renderHook(() => useTradingJournalState({ service }));

    await waitFor(() => expect(service.loadTradingPanelState).toHaveBeenCalledTimes(1));

    await act(async () => {
      await left.result.current.actions.finishJournal("journal-1", "done now", ["discipline"]);
    });

    await waitFor(() => {
      expect(right.result.current.state.journals[0]).toEqual(
        expect.objectContaining({
          status: "done",
          summary: "done now",
          tags: ["discipline"],
        }),
      );
    });
  });

  it("shares recorded events across hook instances", async () => {
    service.loadTradingPanelState.mockResolvedValue({
      journals: [
        {
          id: "journal-1",
          createdAt: "2026-04-18T00:00:00.000Z",
          updatedAt: "2026-04-18T00:00:00.000Z",
          symbol: "ETHUSDT",
          summary: "",
          status: "open",
          events: [],
          images: [],
          tags: [],
        },
      ],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
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

    const left = renderHook(() => useTradingJournalState({ service }));
    const right = renderHook(() => useTradingJournalState({ service }));

    await waitFor(() => expect(service.loadTradingPanelState).toHaveBeenCalledTimes(1));

    await act(async () => {
      await left.result.current.actions.recordEvent("journal-1", {
        type: "log",
        content: "noted",
        priceInput: "",
        quantityInput: "",
      });
    });

    await waitFor(() => {
      expect(right.result.current.state.journals[0]?.events).toHaveLength(1);
    });
  });
});
