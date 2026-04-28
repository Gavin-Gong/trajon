import { beforeEach, describe, expect, it, vi } from "vitest";

const openTrajonDatabaseMock = vi.hoisted(() => vi.fn());

vi.mock("../../database/sqliteStorage", () => ({
  openTrajonDatabase: openTrajonDatabaseMock,
}));

import { runDevToolSql } from "../devToolsSqlRunner";

describe("runDevToolSql", () => {
  beforeEach(() => {
    openTrajonDatabaseMock.mockReset();
  });

  it("uses select for select statements", async () => {
    const select = vi.fn().mockResolvedValue([{ id: 1 }]);
    const execute = vi.fn().mockResolvedValue({ rowsAffected: 0 });
    openTrajonDatabaseMock.mockResolvedValue({ select, execute });

    const result = await runDevToolSql("SELECT * FROM trade_journals");

    expect(select).toHaveBeenCalledWith("SELECT * FROM trade_journals");
    expect(execute).not.toHaveBeenCalled();
    expect(result).toEqual({ kind: "rows", rows: [{ id: 1 }] });
  });

  it("uses select for with queries", async () => {
    const select = vi.fn().mockResolvedValue([{ count: 2 }]);
    const execute = vi.fn().mockResolvedValue({ rowsAffected: 0 });
    openTrajonDatabaseMock.mockResolvedValue({ select, execute });

    const result = await runDevToolSql("WITH counts AS (SELECT 2 AS count) SELECT * FROM counts");

    expect(select).toHaveBeenCalledWith("WITH counts AS (SELECT 2 AS count) SELECT * FROM counts");
    expect(execute).not.toHaveBeenCalled();
    expect(result).toEqual({ kind: "rows", rows: [{ count: 2 }] });
  });

  it("uses select for pragma queries", async () => {
    const select = vi.fn().mockResolvedValue([{ name: "trade_journals" }]);
    const execute = vi.fn().mockResolvedValue({ rowsAffected: 0 });
    openTrajonDatabaseMock.mockResolvedValue({ select, execute });

    const result = await runDevToolSql("PRAGMA table_list");

    expect(select).toHaveBeenCalledWith("PRAGMA table_list");
    expect(execute).not.toHaveBeenCalled();
    expect(result).toEqual({ kind: "rows", rows: [{ name: "trade_journals" }] });
  });

  it("uses execute for update statements", async () => {
    const select = vi.fn().mockResolvedValue([]);
    const execute = vi.fn().mockResolvedValue({ rowsAffected: 1 });
    openTrajonDatabaseMock.mockResolvedValue({ select, execute });

    const result = await runDevToolSql("UPDATE trade_journals SET summary = 'ok'");

    expect(execute).toHaveBeenCalledWith("UPDATE trade_journals SET summary = 'ok'");
    expect(select).not.toHaveBeenCalled();
    expect(result).toEqual({ kind: "execute", result: { rowsAffected: 1 } });
  });

  it("rejects blank sql", async () => {
    const select = vi.fn().mockResolvedValue([]);
    const execute = vi.fn().mockResolvedValue({ rowsAffected: 0 });
    openTrajonDatabaseMock.mockResolvedValue({ select, execute });

    await expect(runDevToolSql("   ")).rejects.toThrow("SQL is required");
    expect(select).not.toHaveBeenCalled();
    expect(execute).not.toHaveBeenCalled();
  });
});
