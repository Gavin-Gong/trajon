import { beforeEach, describe, expect, it, vi } from "vitest";

import { createTradingJournalService } from "../tradingJournalService";

const imagePlatform = {
  persistJournalImages: vi.fn(),
  removeJournalImageFile: vi.fn(),
};

const repository = {
  loadState: vi.fn(),
  insertTradeJournal: vi.fn(),
  appendTradeEvent: vi.fn(),
  insertJournalImages: vi.fn(),
  deleteJournalImage: vi.fn(),
  deleteTradeJournal: vi.fn(),
  updateTradeJournal: vi.fn(),
  writeAppState: vi.fn(),
};

describe("createTradingJournalService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    repository.loadState.mockResolvedValue({
      journals: [],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      panelPinned: true,
    });
  });

  it("loads persisted state from the repository", async () => {
    const service = createTradingJournalService({ repository, imagePlatform });

    const state = await service.loadTradingPanelState();

    expect(repository.loadState).toHaveBeenCalledTimes(1);
    expect(state.panelPinned).toBe(true);
  });

  it("creates a journal and persists it", async () => {
    const service = createTradingJournalService({ repository, imagePlatform });

    const journal = await service.createTradeJournal(" btcusdt ");

    expect(journal.symbol).toBe("BTCUSDT");
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
          summary: "",
          status: "done",
          events: [],
          images: [],
        },
      ],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      panelPinned: true,
    });

    const service = createTradingJournalService({ repository, imagePlatform });
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
          summary: "",
          status: "open",
          events: [],
          images: [],
        },
      ],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [{ symbol: "BTCUSDT", timeframe: "5m" }],
      defaultPresetSymbol: undefined,
      panelPinned: true,
    });

    const service = createTradingJournalService({ repository, imagePlatform });

    await service.finishTradeJournal("journal-1", "  wrapped up  ");
    expect(repository.updateTradeJournal).toHaveBeenCalledWith(
      expect.objectContaining({ status: "done", summary: "wrapped up" }),
    );

    await expect(service.updateConfig({ timerInterval: 0 })).rejects.toThrow();
    await service.setDefaultPresetSymbol("BTCUSDT");
    expect(repository.writeAppState).toHaveBeenCalledWith({ defaultPresetSymbol: "BTCUSDT" });
  });

  it("normalizes journal tags when finishing a journal", async () => {
    repository.loadState.mockResolvedValue({
      journals: [
        {
          id: "journal-1",
          createdAt: "2026-04-18T00:00:00.000Z",
          updatedAt: "2026-04-18T00:00:00.000Z",
          symbol: "BTCUSDT",
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

    const service = createTradingJournalService({ repository, imagePlatform });

    await service.finishTradeJournal("journal-1", "  wrapped up  ", [" breakout ", "FOMO", "breakout", " "]);

    expect(repository.updateTradeJournal).toHaveBeenCalledWith(
      expect.objectContaining({
        status: "done",
        summary: "wrapped up",
        tags: ["breakout", "FOMO"],
      }),
    );
  });

  it("updates a journal symbol without uppercasing it", async () => {
    repository.loadState.mockResolvedValue({
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

    const service = createTradingJournalService({ repository, imagePlatform });

    await service.updateTradeJournalSymbol("journal-1", "SolUsdt");

    expect(repository.updateTradeJournal).toHaveBeenCalledWith(
      expect.objectContaining({
        id: "journal-1",
        symbol: "SolUsdt",
      }),
    );
  });

  it("adds journal images using journalid-timestamp-symbol naming", async () => {
    repository.loadState.mockResolvedValue({
      journals: [
        {
          id: "journal-1",
          createdAt: "2026-04-18T00:00:00.000Z",
          updatedAt: "2026-04-18T00:00:00.000Z",
          symbol: "BTCUSDT",
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

    imagePlatform.persistJournalImages.mockResolvedValue([
      {
        id: "image-1",
        journalId: "journal-1",
        createdAt: "2026-04-18T00:10:00.000Z",
        updatedAt: "2026-04-18T00:10:00.000Z",
        fileName: "journal-1-1713399000000-btcusdt.png",
        relativePath: "trajon/images/journal-1-1713399000000-btcusdt.png",
        sortOrder: 0,
      },
    ]);

    const service = createTradingJournalService({ repository, imagePlatform });
    const images = await service.addJournalImages("journal-1", ["/tmp/chart.png"]);

    expect(imagePlatform.persistJournalImages).toHaveBeenCalledWith(
      expect.objectContaining({ id: "journal-1", symbol: "BTCUSDT" }),
      ["/tmp/chart.png"],
    );
    expect(repository.insertJournalImages).toHaveBeenCalledWith("journal-1", [
      expect.objectContaining({ fileName: "journal-1-1713399000000-btcusdt.png" }),
    ]);
    expect(images[0]?.fileName).toBe("journal-1-1713399000000-btcusdt.png");
  });

  it("removes journal images from both sqlite and disk", async () => {
    repository.loadState.mockResolvedValue({
      journals: [
        {
          id: "journal-1",
          createdAt: "2026-04-18T00:00:00.000Z",
          updatedAt: "2026-04-18T00:00:00.000Z",
          symbol: "BTCUSDT",
          summary: "",
          status: "open",
          events: [],
          images: [
            {
              id: "image-1",
              journalId: "journal-1",
              createdAt: "2026-04-18T00:10:00.000Z",
              updatedAt: "2026-04-18T00:10:00.000Z",
              fileName: "journal-1-1713399000000-btcusdt.png",
              relativePath: "trajon/images/journal-1-1713399000000-btcusdt.png",
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

    const service = createTradingJournalService({ repository, imagePlatform });
    await service.removeJournalImage("journal-1", "image-1");

    expect(repository.deleteJournalImage).toHaveBeenCalledWith("image-1");
    expect(imagePlatform.removeJournalImageFile).toHaveBeenCalledWith(
      "trajon/images/journal-1-1713399000000-btcusdt.png",
    );
  });
});
