import { openTrajonDatabase } from "../../database/sqliteStorage";
import type { TradeEvent, TradeJournal } from "../../domain/tradingJournalDomain";
import type { TradingPanelState } from "../../model/storage";
import { runTradingJournalMigrations } from "./tradingJournalMigrations";
import {
  appConfigValueSchema,
  tradeEventRowSchema,
  tradeJournalRowSchema,
  tradingJournalStateSchema,
} from "./tradingJournalSchemas";

type SqlDatabase = {
  execute: (sql: string, bindValues?: unknown[]) => Promise<unknown>;
  select: <T>(sql: string, bindValues?: unknown[]) => Promise<T[]>;
};

type AppStateUpdate = Partial<Pick<TradingPanelState, "config" | "pairPresets" | "defaultPresetSymbol" | "panelPinned">>;

const DEFAULT_STATE: TradingPanelState = {
  journals: [],
  config: { timerEnabled: false, timerInterval: 300 },
  pairPresets: [],
  defaultPresetSymbol: undefined,
  panelPinned: true,
};

export function createTradingJournalRepository() {
  let databasePromise: Promise<SqlDatabase> | undefined;

  async function getDatabase(): Promise<SqlDatabase> {
    if (!databasePromise) {
        databasePromise = openTrajonDatabase().then(async (db) => {
        await runTradingJournalMigrations(db as SqlDatabase);
        return db as SqlDatabase;
      });
    }

    return databasePromise;
  }

  return {
    async loadState(): Promise<TradingPanelState> {
      const db = await getDatabase();
      const journalRows = await db.select<Record<string, unknown>>(
        "SELECT id, created_at, updated_at, symbol, market, summary, status FROM trade_journals ORDER BY created_at DESC",
      );
      const eventRows = await db.select<Record<string, unknown>>(
        "SELECT id, journal_id, created_at, updated_at, timestamp, type, content, price, quantity FROM trade_events ORDER BY timestamp DESC",
      );
      const appConfigRows = await db.select<{ key: string; value: string }>(
        "SELECT key, value FROM app_config",
      );

      const journals = tradeJournalRowSchema.array().parse(journalRows).map((row) => ({
        id: row.id,
        createdAt: row.created_at,
        updatedAt: row.updated_at,
        symbol: row.symbol,
        market: row.market,
        summary: row.summary,
        status: row.status,
        events: [],
      }));

      const parsedEvents = tradeEventRowSchema.array().parse(eventRows);
      const eventsByJournalId = new Map<string, TradeEvent[]>();

      for (const row of parsedEvents) {
        const event: TradeEvent = {
          id: row.id,
          createdAt: row.created_at,
          updatedAt: row.updated_at,
          timestamp: row.timestamp,
          type: row.type,
          content: row.content,
          price: row.price ?? undefined,
          quantity: row.quantity ?? undefined,
        };
        const existing = eventsByJournalId.get(row.journal_id) ?? [];
        existing.push(event);
        eventsByJournalId.set(row.journal_id, existing);
      }

      const configPatch = parseAppConfigRows(appConfigRows);
      return tradingJournalStateSchema.parse({
        ...DEFAULT_STATE,
        ...configPatch,
        journals: journals.map((journal) => ({
          ...journal,
          events: eventsByJournalId.get(journal.id) ?? [],
        })),
      });
    },

    async insertTradeJournal(journal: TradeJournal): Promise<void> {
      const db = await getDatabase();
      await db.execute(
        "INSERT INTO trade_journals (id, created_at, updated_at, symbol, market, summary, status) VALUES ($1, $2, $3, $4, $5, $6, $7)",
        [
          journal.id,
          journal.createdAt,
          journal.updatedAt,
          journal.symbol,
          journal.market,
          journal.summary,
          journal.status,
        ],
      );
    },

    async appendTradeEvent(journalId: string, event: TradeEvent): Promise<void> {
      const db = await getDatabase();
      await db.execute(
        "INSERT INTO trade_events (id, journal_id, created_at, updated_at, timestamp, type, content, price, quantity) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
        [
          event.id,
          journalId,
          event.createdAt,
          event.updatedAt,
          event.timestamp,
          event.type,
          event.content,
          event.price ?? null,
          event.quantity ?? null,
        ],
      );
      await db.execute(
        "UPDATE trade_journals SET updated_at = $2 WHERE id = $1",
        [journalId, event.updatedAt],
      );
    },

    async updateTradeJournal(journal: TradeJournal): Promise<void> {
      const db = await getDatabase();
      await db.execute(
        "UPDATE trade_journals SET updated_at = $2, summary = $3, status = $4, symbol = $5, market = $6 WHERE id = $1",
        [journal.id, journal.updatedAt, journal.summary, journal.status, journal.symbol, journal.market],
      );
    },

    async writeAppState(state: AppStateUpdate): Promise<void> {
      const db = await getDatabase();
      const timestamp = new Date().toISOString();
      const parsed = appConfigValueSchema.parse(state);

      for (const [key, value] of Object.entries(parsed)) {
        if (value === undefined) {
          continue;
        }

        await db.execute(
          "INSERT OR REPLACE INTO app_config (key, value, updated_at) VALUES ($1, $2, $3)",
          [key, JSON.stringify(value), timestamp],
        );
      }
    },
  };
}

function parseAppConfigRows(rows: { key: string; value: string }[]): AppStateUpdate {
  const raw: Record<string, unknown> = {};

  for (const row of rows) {
    raw[row.key] = JSON.parse(row.value);
  }

  return appConfigValueSchema.parse(raw);
}
