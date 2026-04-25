import { describe, expect, it } from "vitest";

import type {
  TradeEvent,
  TradeEventType,
  JournalImage,
  TradeJournal,
  TradeJournalStatus,
} from "./model";

describe("journal model exports", () => {
  it("defines the journal and trade event data models", () => {
    const status: TradeJournalStatus = "open";
    const eventType: TradeEventType = "entry";
    const event: TradeEvent = {
      id: "event-1",
      createdAt: "2026-04-18T00:00:00.000Z",
      updatedAt: "2026-04-18T00:00:00.000Z",
      timestamp: "2026-04-18T00:00:00.000Z",
      type: eventType,
      content: "Initial entry",
      price: 101.25,
      quantity: 2,
    };
    const journal: TradeJournal = {
      id: "journal-1",
      createdAt: "2026-04-18T00:00:00.000Z",
      updatedAt: "2026-04-18T01:00:00.000Z",
      symbol: "BTCUSDT",
      market: "BINANCE",
      summary: "Breakout setup",
      tags: [],
      status,
      events: [event],
      images: [
        {
          id: "image-1",
          journalId: "journal-1",
          createdAt: "2026-04-18T00:10:00.000Z",
          updatedAt: "2026-04-18T00:10:00.000Z",
          fileName: "journal-1-1713399000000-btcusdt.png",
          relativePath: "images/journal-1-1713399000000-btcusdt.png",
          sortOrder: 0,
        },
      ],
    };

    expect(journal.status).toBe("open");
    expect(journal.updatedAt).toBe("2026-04-18T01:00:00.000Z");
    expect(journal.events[0].createdAt).toBe("2026-04-18T00:00:00.000Z");
    expect(journal.events[0]).toEqual(event);
    expect(journal.images[0]?.journalId).toBe("journal-1");
  });

  it("defines journal image metadata for persisted review screenshots", () => {
    const image: JournalImage = {
      id: "image-1",
      journalId: "journal-1",
      createdAt: "2026-04-18T00:10:00.000Z",
      updatedAt: "2026-04-18T00:10:00.000Z",
      fileName: "journal-1-1713399000000-btcusdt.png",
      relativePath: "images/journal-1-1713399000000-btcusdt.png",
      sortOrder: 0,
    };

    expect(image.fileName).toContain("journal-1-");
    expect(image.relativePath).toBe("images/journal-1-1713399000000-btcusdt.png");
  });

  it("supports the full trade event type set from the source model", () => {
    const eventTypes: TradeEventType[] = ["entry", "add", "reduce", "exit", "log", "plan"];

    expect(eventTypes).toEqual(["entry", "add", "reduce", "exit", "log", "plan"]);
  });
});
