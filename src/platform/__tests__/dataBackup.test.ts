import { beforeEach, describe, expect, it, vi } from "vitest";

const pathMock = vi.hoisted(() => ({
  appDataDir: vi.fn(),
  join: vi.fn(),
}));

const fsMock = vi.hoisted(() => ({
  readDir: vi.fn(),
}));

const sqliteMock = vi.hoisted(() => ({
  BACKUP_DIRECTORY: "databases/backups",
  backupTrajonDatabase: vi.fn(),
  restoreTrajonDatabase: vi.fn(),
}));

vi.mock("@tauri-apps/api/path", () => pathMock);
vi.mock("@tauri-apps/plugin-fs", () => fsMock);
vi.mock("../../database/sqliteStorage", () => sqliteMock);

import {
  buildDatabaseBackupFileName,
  createDatabaseBackup,
  isDatabaseBackupAvailable,
  listDatabaseBackups,
  restoreDatabaseBackup,
} from "../dataBackup";

describe("dataBackup", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useRealTimers();
    delete window.__TAURI_INTERNALS__;
    pathMock.appDataDir.mockResolvedValue("/Users/zen/Library/Application Support/dev.trajon");
    pathMock.join.mockImplementation(async (...parts: string[]) => parts.join("/"));
  });

  it("reports backup support only in Tauri runtime", () => {
    expect(isDatabaseBackupAvailable()).toBe(false);
    window.__TAURI_INTERNALS__ = {};
    expect(isDatabaseBackupAvailable()).toBe(true);
  });

  it("builds a timestamped backup file name", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-04-26T09:08:07.000Z"));

    expect(buildDatabaseBackupFileName()).toBe("trajon-backup-20260426-090807.db");
  });

  it("creates a backup with the generated file name", async () => {
    sqliteMock.backupTrajonDatabase.mockResolvedValue("databases/backups/trajon-backup-20260426-090807.db");
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-04-26T09:08:07.000Z"));

    await expect(createDatabaseBackup()).resolves.toEqual({
      fileName: "trajon-backup-20260426-090807.db",
      relativePath: "databases/backups/trajon-backup-20260426-090807.db",
    });
  });

  it("lists backup files from the app backup directory", async () => {
    fsMock.readDir.mockResolvedValue([
      {
        name: "trajon-backup-20260426-090807.db",
        isFile: true,
      },
      {
        name: "notes.txt",
        isFile: true,
      },
      {
        name: "older.db",
        isFile: true,
      },
      {
        name: "nested",
        isFile: false,
      },
    ]);

    await expect(listDatabaseBackups()).resolves.toEqual([
      {
        fileName: "trajon-backup-20260426-090807.db",
        absolutePath: "/Users/zen/Library/Application Support/dev.trajon/databases/backups/trajon-backup-20260426-090807.db",
      },
      {
        fileName: "older.db",
        absolutePath: "/Users/zen/Library/Application Support/dev.trajon/databases/backups/older.db",
      },
    ]);
  });

  it("restores a selected backup by file name", async () => {
    await restoreDatabaseBackup("trajon-backup-20260426-090807.db");

    expect(sqliteMock.restoreTrajonDatabase).toHaveBeenCalledWith("trajon-backup-20260426-090807.db");
  });
});
