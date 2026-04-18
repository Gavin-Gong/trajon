import { beforeEach, describe, expect, it, vi } from "vitest";

import { createTradingPanelService } from "./tradingPanelPersistence/tradingPanelService";

const repository = {
  loadState: vi.fn(),
  insertTradeJournal: vi.fn(),
  appendTradeEvent: vi.fn(),
  updateTradeJournal: vi.fn(),
  writeAppState: vi.fn(),
};

describe("createTradingPanelService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    repository.loadState.mockResolvedValue({
      journals: [],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      chart: { symbol: "BINANCE:BTCUSDT", interval: "60" },
      panelPinned: true,
    });
  });

  it("loads persisted state from the repository", async () => {
    const service = createTradingPanelService({ repository });

    const state = await service.loadTradingPanelState();

    expect(repository.loadState).toHaveBeenCalledTimes(1);
    expect(state.chart.symbol).toBe("BINANCE:BTCUSDT");
  });

  it("creates a journal and persists it", async () => {
    const service = createTradingPanelService({ repository });

    const journal = await service.createTradeJournal(" btcusdt ", " binance ");

    expect(journal.symbol).toBe("BTCUSDT");
    expect(journal.market).toBe("binance");
    expect(repository.insertTradeJournal).toHaveBeenCalledWith(expect.objectContaining({ symbol: "BTCUSDT" }));
  });

  it("rejects invalid event transitions and persists valid events", async () => {
    repository.loadState.mockResolvedValueOnce({
      journals: [
        {
          id: "journal-1",
          createdAt: "2026-04-18T00:00:00.000Z",
          updatedAt: "2026-04-18T00:00:00.000Z",
          symbol: "BTCUSDT",
          market: "BINANCE",
          summary: "",
          status: "done",
          events: [],
        },
      ],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      chart: { symbol: "BINANCE:BTCUSDT", interval: "60" },
      panelPinned: true,
    });

    const service = createTradingPanelService({ repository });
    const failed = await service.createTradeEvent("journal-1", {
      type: "entry",
      content: "Open long",
      priceInput: "100",
      quantityInput: "1",
    });

    expect(failed).toEqual({ ok: false, error: "doneOnlyLogs" });
    expect(repository.appendTradeEvent).not.toHaveBeenCalled();

    repository.loadState.mockResolvedValueOnce({
      journals: [
        {
          id: "journal-2",
          createdAt: "2026-04-18T00:00:00.000Z",
          updatedAt: "2026-04-18T00:00:00.000Z",
          symbol: "BTCUSDT",
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

    const success = await service.createTradeEvent("journal-2", {
      type: "entry",
      content: "Open long",
      priceInput: "100",
      quantityInput: "1",
    });

    expect(success.ok).toBe(true);
    expect(repository.appendTradeEvent).toHaveBeenCalledTimes(1);
  });

  it("finishes a journal and validates app state updates", async () => {
    repository.loadState.mockResolvedValue({
      journals: [
        {
          id: "journal-1",
          createdAt: "2026-04-18T00:00:00.000Z",
          updatedAt: "2026-04-18T00:00:00.000Z",
          symbol: "BTCUSDT",
          market: "BINANCE",
          summary: "",
          status: "open",
          events: [],
        },
      ],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [{ symbol: "BTCUSDT", timeframe: "5m" }],
      defaultPresetSymbol: undefined,
      chart: { symbol: "BINANCE:BTCUSDT", interval: "60" },
      panelPinned: true,
    });

    const service = createTradingPanelService({ repository });

    await service.finishTradeJournal("journal-1", "  wrapped up  ");
    expect(repository.updateTradeJournal).toHaveBeenCalledWith(
      expect.objectContaining({ status: "done", summary: "wrapped up" }),
    );

    await expect(service.updateConfig({ timerInterval: 0 })).rejects.toThrow();
    await service.setDefaultPresetSymbol("BTCUSDT");
    expect(repository.writeAppState).toHaveBeenCalledWith({ defaultPresetSymbol: "BTCUSDT" });
  });
});
