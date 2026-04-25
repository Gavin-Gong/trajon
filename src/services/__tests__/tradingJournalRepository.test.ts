import { beforeEach, describe, expect, it, vi } from "vitest";

import type { TradeEvent, TradeJournal } from "../../domain/journal";
import { createTradingJournalRepository } from "../tradingJournalRepository";
import { runTradingJournalMigrations } from "../tradingJournalMigrations";

const openTrajonDatabaseMock = vi.hoisted(() => vi.fn());

vi.mock("../../database/sqliteStorage", () => ({
  openTrajonDatabase: openTrajonDatabaseMock,
}));

describe("runTradingJournalMigrations", () => {
  it("creates the base trading panel tables and indexes", async () => {
    const execute = vi.fn().mockResolvedValue(undefined);
    const select = vi.fn().mockResolvedValue([]);
    const db = { execute, select };

    await runTradingJournalMigrations(db as never);

    expect(
      execute.mock.calls.some(([sql]) => String(sql).includes("CREATE TABLE IF NOT EXISTS trade_journals")),
    ).toBe(true);
    expect(
      execute.mock.calls.some(([sql]) => String(sql).includes("CREATE TABLE IF NOT EXISTS trade_events")),
    ).toBe(true);
    expect(
      execute.mock.calls.some(([sql]) => String(sql).includes("CREATE TABLE IF NOT EXISTS app_config")),
    ).toBe(true);
  });
});

describe("createTradingJournalRepository", () => {
  beforeEach(() => {
    openTrajonDatabaseMock.mockReset();
  });

  it("loads default state from an empty database", async () => {
    const db = {
      execute: vi.fn().mockResolvedValue(undefined),
      select: vi
        .fn()
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([]),
    };
    openTrajonDatabaseMock.mockResolvedValue(db);

    const repository = createTradingJournalRepository();
    const state = await repository.loadState();

    expect(state).toEqual({
      journals: [],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      panelPinned: true,
    });
  });

  it("loads journal images and groups them under each journal", async () => {
    const db = {
      execute: vi.fn().mockResolvedValue(undefined),
      select: vi
        .fn()
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([
          {
            id: "journal-1",
            created_at: "2026-04-18T00:00:00.000Z",
            updated_at: "2026-04-18T00:00:00.000Z",
            symbol: "BTCUSDT",
            market: "BINANCE",
            summary: "",
            tags: '["breakout","trend"]',
            status: "open",
          },
        ])
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([
          {
            id: "image-2",
            journal_id: "journal-1",
            created_at: "2026-04-18T00:02:00.000Z",
            updated_at: "2026-04-18T00:02:00.000Z",
            file_name: "journal-1-1713398520000-btcusdt.png",
            relative_path: "images/journal-1-1713398520000-btcusdt.png",
            sort_order: 1,
          },
          {
            id: "image-1",
            journal_id: "journal-1",
            created_at: "2026-04-18T00:01:00.000Z",
            updated_at: "2026-04-18T00:01:00.000Z",
            file_name: "journal-1-1713398460000-btcusdt.png",
            relative_path: "images/journal-1-1713398460000-btcusdt.png",
            sort_order: 0,
          },
        ])
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([]),
    };
    openTrajonDatabaseMock.mockResolvedValue(db);

    const repository = createTradingJournalRepository();
    const state = await repository.loadState();

    expect(state.journals[0]?.images).toEqual([
      {
        id: "image-1",
        journalId: "journal-1",
        createdAt: "2026-04-18T00:01:00.000Z",
        updatedAt: "2026-04-18T00:01:00.000Z",
        fileName: "journal-1-1713398460000-btcusdt.png",
        relativePath: "images/journal-1-1713398460000-btcusdt.png",
        sortOrder: 0,
      },
      {
        id: "image-2",
        journalId: "journal-1",
        createdAt: "2026-04-18T00:02:00.000Z",
        updatedAt: "2026-04-18T00:02:00.000Z",
        fileName: "journal-1-1713398520000-btcusdt.png",
        relativePath: "images/journal-1-1713398520000-btcusdt.png",
        sortOrder: 1,
      },
    ]);
    expect(state.journals[0]?.tags).toEqual(["breakout", "trend"]);
  });

  it("persists journal, event, and app state writes", async () => {
    const db = {
      execute: vi.fn().mockResolvedValue(undefined),
      select: vi
        .fn()
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([]),
    };
    openTrajonDatabaseMock.mockResolvedValue(db);

    const repository = createTradingJournalRepository();
    const journal: TradeJournal = {
      id: "journal-1",
      createdAt: "2026-04-18T00:00:00.000Z",
      updatedAt: "2026-04-18T00:00:00.000Z",
      symbol: "BTCUSDT",
      market: "BINANCE",
      summary: "",
      tags: ["breakout"],
      status: "open",
      events: [],
      images: [],
    };
    const event: TradeEvent = {
      id: "event-1",
      createdAt: "2026-04-18T00:01:00.000Z",
      updatedAt: "2026-04-18T00:01:00.000Z",
      timestamp: "2026-04-18T00:01:00.000Z",
      type: "log",
      content: "Noted breakout",
    };

    await repository.insertTradeJournal(journal);
    await repository.appendTradeEvent(journal.id, event);
    await repository.writeAppState({
      config: { timerEnabled: true, timerInterval: 600 },
      pairPresets: [{ symbol: "BTCUSDT", timeframe: "5m" }],
      defaultPresetSymbol: "BTCUSDT",
      panelPinned: false,
    });

    expect(db.execute).toHaveBeenCalledWith(
      expect.stringContaining("INSERT INTO trade_journals"),
      expect.any(Array),
    );
    expect(db.execute).toHaveBeenCalledWith(
      expect.stringContaining("INSERT INTO trade_events"),
      expect.any(Array),
    );
    expect(db.execute).toHaveBeenCalledWith(
      expect.stringContaining("UPDATE trade_journals SET updated_at"),
      expect.any(Array),
    );
    expect(db.execute).toHaveBeenCalledWith(
      expect.stringContaining("INSERT OR REPLACE INTO app_config"),
      expect.any(Array),
    );
  });

  it("persists and removes journal images", async () => {
    const db = {
      execute: vi.fn().mockResolvedValue(undefined),
      select: vi
        .fn()
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([]),
    };
    openTrajonDatabaseMock.mockResolvedValue(db);

    const repository = createTradingJournalRepository();

    await repository.insertJournalImages("journal-1", [
      {
        id: "image-1",
        journalId: "journal-1",
        createdAt: "2026-04-18T00:01:00.000Z",
        updatedAt: "2026-04-18T00:01:00.000Z",
        fileName: "journal-1-1713398460000-btcusdt.png",
        relativePath: "images/journal-1-1713398460000-btcusdt.png",
        sortOrder: 0,
      },
    ]);
    await repository.deleteJournalImage("image-1");

    expect(db.execute).toHaveBeenCalledWith(
      expect.stringContaining("INSERT INTO journal_images"),
      expect.any(Array),
    );
    expect(db.execute).toHaveBeenCalledWith(
      expect.stringContaining("DELETE FROM journal_images WHERE id = $1"),
      ["image-1"],
    );
  });

  it("rejects invalid journal rows from sql", async () => {
    const db = {
      execute: vi.fn().mockResolvedValue(undefined),
      select: vi
        .fn()
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([
          {
            id: "journal-1",
            created_at: "2026-04-18T00:00:00.000Z",
            updated_at: "2026-04-18T00:00:00.000Z",
            symbol: "BTCUSDT",
            market: "BINANCE",
            summary: "",
            status: "broken",
          },
        ])
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([]),
    };
    openTrajonDatabaseMock.mockResolvedValue(db);

    const repository = createTradingJournalRepository();

    await expect(repository.loadState()).rejects.toThrow();
  });
});
