type SqlExecutor = {
  execute: (sql: string, bindValues?: unknown[]) => Promise<unknown>;
  select: <T>(sql: string, bindValues?: unknown[]) => Promise<T[]>;
};

type Migration = {
  version: number;
  statements: string[];
};

const migrations: Migration[] = [
  {
    version: 1,
    statements: [
      `CREATE TABLE IF NOT EXISTS trade_journals (
        id TEXT PRIMARY KEY,
        created_at TEXT NOT NULL,
        updated_at TEXT NOT NULL,
        symbol TEXT NOT NULL,
        market TEXT NOT NULL,
        summary TEXT NOT NULL,
        status TEXT NOT NULL
      )`,
      `CREATE TABLE IF NOT EXISTS trade_events (
        id TEXT PRIMARY KEY,
        journal_id TEXT NOT NULL,
        created_at TEXT NOT NULL,
        updated_at TEXT NOT NULL,
        timestamp TEXT NOT NULL,
        type TEXT NOT NULL,
        content TEXT NOT NULL,
        price REAL,
        quantity REAL,
        FOREIGN KEY (journal_id) REFERENCES trade_journals(id) ON DELETE CASCADE
      )`,
      `CREATE TABLE IF NOT EXISTS app_config (
        key TEXT PRIMARY KEY,
        value TEXT NOT NULL,
        updated_at TEXT NOT NULL
      )`,
      "CREATE INDEX IF NOT EXISTS idx_trade_journals_status ON trade_journals(status)",
      "CREATE INDEX IF NOT EXISTS idx_trade_journals_created_at ON trade_journals(created_at)",
      "CREATE INDEX IF NOT EXISTS idx_trade_events_journal_id ON trade_events(journal_id)",
      "CREATE INDEX IF NOT EXISTS idx_trade_events_timestamp ON trade_events(timestamp)",
    ],
  },
];

export async function runTradingJournalMigrations(db: SqlExecutor): Promise<void> {
  await db.execute(`CREATE TABLE IF NOT EXISTS migrations (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL
  )`);

  const applied = await db.select<{ version: number }>("SELECT version FROM migrations");
  const appliedVersions = new Set(applied.map((row) => row.version));

  for (const migration of migrations) {
    if (appliedVersions.has(migration.version)) {
      continue;
    }

    for (const statement of migration.statements) {
      await db.execute(statement);
    }

    await db.execute(
      "INSERT INTO migrations (version, applied_at) VALUES ($1, $2)",
      [migration.version, new Date().toISOString()],
    );
  }
}
