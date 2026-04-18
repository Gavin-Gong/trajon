import { beforeEach, describe, expect, it, vi } from "vitest";

import type { TradeEvent, TradeJournal } from "../../../domain/tradingJournalDomain";
import { createTradingJournalRepository } from "../tradingJournalRepository";
import { runTradingJournalMigrations } from "../tradingJournalMigrations";

const openTradingFlowDatabaseMock = vi.hoisted(() => vi.fn());

vi.mock("../../../database/sqliteStorage", () => ({
  openTradingFlowDatabase: openTradingFlowDatabaseMock,
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
    openTradingFlowDatabaseMock.mockReset();
  });

  it("loads default state from an empty database", async () => {
    const db = {
      execute: vi.fn().mockResolvedValue(undefined),
      select: vi
        .fn()
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([]),
    };
    openTradingFlowDatabaseMock.mockResolvedValue(db);

    const repository = createTradingJournalRepository();
    const state = await repository.loadState();

    expect(state).toEqual({
      journals: [],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      chart: { symbol: "BINANCE:BTCUSDT", interval: "60" },
      panelPinned: true,
    });
  });

  it("persists journal, event, and app state writes", async () => {
    const db = {
      execute: vi.fn().mockResolvedValue(undefined),
      select: vi
        .fn()
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([])
        .mockResolvedValueOnce([]),
    };
    openTradingFlowDatabaseMock.mockResolvedValue(db);

    const repository = createTradingJournalRepository();
    const journal: TradeJournal = {
      id: "journal-1",
      createdAt: "2026-04-18T00:00:00.000Z",
      updatedAt: "2026-04-18T00:00:00.000Z",
      symbol: "BTCUSDT",
      market: "BINANCE",
      summary: "",
      status: "open",
      events: [],
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
      chart: { symbol: "BINANCE:ETHUSDT", interval: "240" },
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
        .mockResolvedValueOnce([]),
    };
    openTradingFlowDatabaseMock.mockResolvedValue(db);

    const repository = createTradingJournalRepository();

    await expect(repository.loadState()).rejects.toThrow();
  });
});
