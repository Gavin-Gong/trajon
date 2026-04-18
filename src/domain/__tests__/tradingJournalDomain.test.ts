import { describe, expect, it } from "vitest";

import {
  availableEventTypes,
  composeTradeEvent,
  createJournal,
  currentJournal,
  defaultCalendarSelection,
  realizedPnlAnalytics,
  groupJournalsByDay,
  netPosition,
  visibleMonthDays,
  type TradeEventDraft,
} from "../tradingJournalDomain";

function buildDraft(overrides: Partial<TradeEventDraft> = {}): TradeEventDraft {
  return {
    type: "log",
    content: "Observed absorption at VWAP",
    priceInput: "",
    quantityInput: "",
    ...overrides,
  };
}

describe("currentJournal", () => {
  it("prefers open journals over done journals", () => {
    const doneJournal = createJournal({ symbol: "ETHUSDT", status: "done" });
    const openJournal = createJournal({ symbol: "BTCUSDT", status: "open" });

    expect(currentJournal([doneJournal, openJournal])?.id).toBe(openJournal.id);
  });

  it("uses symbol then id as stable tie breakers", () => {
    const beta = createJournal({ id: "b", symbol: "SOL" });
    const alpha = createJournal({ id: "a", symbol: "BTC" });
    const alphaLater = createJournal({ id: "z", symbol: "BTC" });

    expect(currentJournal([beta, alphaLater, alpha])?.id).toBe("a");
  });

  it("returns undefined when all journals are already done", () => {
    const doneAlpha = createJournal({ id: "a", symbol: "BTC", status: "done" });
    const doneBeta = createJournal({ id: "b", symbol: "ETH", status: "done" });

    expect(currentJournal([doneBeta, doneAlpha])).toBeUndefined();
  });
});

describe("availableEventTypes", () => {
  it("allows only log and entry on a flat open journal", () => {
    expect(availableEventTypes("open", 0)).toEqual(["log", "entry"]);
  });

  it("allows only log on a finished journal", () => {
    expect(availableEventTypes("done", 3)).toEqual(["log"]);
  });
});

describe("composeTradeEvent", () => {
  it("creates a log when text is present", () => {
    const result = composeTradeEvent(buildDraft(), 0, "open");

    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.event.type).toBe("log");
      expect(result.event.content).toBe("Observed absorption at VWAP");
      expect(result.event.createdAt).toBe(result.event.timestamp);
      expect(result.event.updatedAt).toBe(result.event.timestamp);
    }
  });

  it("converts a reduce that closes the position into an exit", () => {
    const result = composeTradeEvent(
      buildDraft({ type: "reduce", quantityInput: "-2", priceInput: "61325.5" }),
      2,
      "open",
    );

    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.event.type).toBe("exit");
      expect(result.event.quantity).toBe(-2);
      expect(result.event.price).toBe(61325.5);
    }
  });

  it("rejects adds that do not match the current position direction", () => {
    const result = composeTradeEvent(
      buildDraft({ type: "add", quantityInput: "-1" }),
      2,
      "open",
    );

    expect(result).toEqual({ ok: false, error: "wrongDirection" });
  });

  it("rejects reductions larger than the remaining position", () => {
    const result = composeTradeEvent(
      buildDraft({ type: "reduce", quantityInput: "-3" }),
      2,
      "open",
    );

    expect(result).toEqual({ ok: false, error: "exceedsPosition" });
  });
});

describe("calendar helpers", () => {
  it("groups same-day journals in chronological order", () => {
    const morning = createJournal({
      id: "2",
      symbol: "ETH",
      createdAt: "2026-04-04T09:00:00.000Z",
    });
    const early = createJournal({
      id: "1",
      symbol: "BTC",
      createdAt: "2026-04-04T08:00:00.000Z",
    });

    const groups = groupJournalsByDay([morning, early]);

    expect(groups[0]?.journals.map((journal) => journal.id)).toEqual(["1", "2"]);
  });

  it("selects the first journal within the displayed month by created time", () => {
    const march = createJournal({ id: "march", createdAt: "2026-03-31T23:00:00.000Z" });
    const aprilFirst = createJournal({ id: "april-1", createdAt: "2026-04-01T08:00:00.000Z" });
    const aprilSecond = createJournal({ id: "april-2", createdAt: "2026-04-02T08:00:00.000Z" });

    expect(
      defaultCalendarSelection(new Date("2026-04-12T12:00:00.000Z"), [aprilSecond, march, aprilFirst])?.id,
    ).toBe("april-1");
  });

  it("builds a complete visible month grid", () => {
    const days = visibleMonthDays(new Date("2026-04-12T12:00:00.000Z"));

    expect(days.length).toBeGreaterThanOrEqual(35);
    expect(days.some((day) => day.isInDisplayedMonth)).toBe(true);
  });
});

describe("createJournal", () => {
  it("defaults createdAt and updatedAt together", () => {
    const journal = createJournal({ symbol: "BTCUSDT" });

    expect(journal.createdAt).toBeTruthy();
    expect(journal.updatedAt).toBe(journal.createdAt);
  });
});

describe("netPosition", () => {
  it("sums event quantities and ignores empty quantities", () => {
    const entry = composeTradeEvent(buildDraft({ type: "entry", quantityInput: "2" }), 0, "open");
    const log = composeTradeEvent(buildDraft({ type: "log" }), 2, "open");
    const reduce = composeTradeEvent(buildDraft({ type: "reduce", quantityInput: "-0.5" }), 2, "open");

    expect(entry.ok && log.ok && reduce.ok).toBe(true);
    if (entry.ok && log.ok && reduce.ok) {
      expect(netPosition([entry.event, log.event, reduce.event])).toBe(1.5);
    }
  });
});

describe("realizedPnlAnalytics", () => {
  it("builds cumulative equity and daily pnl from completed journals", () => {
    const journal = createJournal({
      id: "journal-1",
      symbol: "BTCUSDT",
      market: "BINANCE",
      status: "done",
      events: [
        {
          id: "entry-1",
          createdAt: "2026-04-18T09:00:00.000Z",
          updatedAt: "2026-04-18T09:00:00.000Z",
          timestamp: "2026-04-18T09:00:00.000Z",
          type: "entry",
          content: "Open starter",
          price: 100,
          quantity: 1,
        },
        {
          id: "add-1",
          createdAt: "2026-04-18T10:00:00.000Z",
          updatedAt: "2026-04-18T10:00:00.000Z",
          timestamp: "2026-04-18T10:00:00.000Z",
          type: "add",
          content: "Add winner",
          price: 110,
          quantity: 1,
        },
        {
          id: "reduce-1",
          createdAt: "2026-04-18T11:00:00.000Z",
          updatedAt: "2026-04-18T11:00:00.000Z",
          timestamp: "2026-04-18T11:00:00.000Z",
          type: "reduce",
          content: "Take partial",
          price: 120,
          quantity: -1,
        },
        {
          id: "exit-1",
          createdAt: "2026-04-19T09:30:00.000Z",
          updatedAt: "2026-04-19T09:30:00.000Z",
          timestamp: "2026-04-19T09:30:00.000Z",
          type: "exit",
          content: "Close rest",
          price: 90,
          quantity: -1,
        },
      ],
    });

    const analytics = realizedPnlAnalytics([journal]);

    expect(analytics.realizedTotal).toBe(0);
    expect(analytics.equityCurve).toEqual([
      { timestamp: "2026-04-18T11:00:00.000Z", value: 15 },
      { timestamp: "2026-04-19T09:30:00.000Z", value: 0 },
    ]);
    expect(analytics.dailyPnl).toEqual([
      { dayKey: "2026-04-18", value: 15 },
      { dayKey: "2026-04-19", value: -15 },
    ]);
    expect(analytics.journalCount).toBe(1);
  });
});
