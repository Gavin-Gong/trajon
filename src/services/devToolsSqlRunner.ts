import { openTrajonDatabase } from "../database/sqliteStorage";

type SqlDatabase = {
  execute: (sql: string, bindValues?: unknown[]) => Promise<unknown>;
  select: <T>(sql: string, bindValues?: unknown[]) => Promise<T[]>;
};

export type DevToolSqlResult =
  | { kind: "rows"; rows: unknown[] }
  | { kind: "execute"; result: unknown };

const SELECT_PREFIXES = ["select", "with", "pragma"];

export async function runDevToolSql(sql: string): Promise<DevToolSqlResult> {
  const statement = sql.trim();
  if (!statement) {
    throw new Error("SQL is required");
  }

  const db = (await openTrajonDatabase()) as SqlDatabase;
  const normalizedPrefix = statement.match(/^[a-zA-Z]+/)?.[0]?.toLowerCase();

  if (normalizedPrefix && SELECT_PREFIXES.includes(normalizedPrefix)) {
    const rows = await db.select<unknown>(statement);
    return { kind: "rows", rows };
  }

  const result = await db.execute(statement);
  return { kind: "execute", result };
}
