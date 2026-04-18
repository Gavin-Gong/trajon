import { beforeEach, describe, expect, it, vi } from "vitest";
import { BaseDirectory, copyFile, exists, mkdir, remove } from "@tauri-apps/plugin-fs";
import Database from "@tauri-apps/plugin-sql";

const fsMock = vi.hoisted(() => ({
  BaseDirectory: {
    AppData: "AppData",
  },
  copyFile: vi.fn(),
  exists: vi.fn(),
  mkdir: vi.fn(),
  remove: vi.fn(),
}));

const closeMock = vi.hoisted(() => vi.fn());
const loadMock = vi.hoisted(() => vi.fn());

vi.mock("@tauri-apps/plugin-fs", () => fsMock);
vi.mock("@tauri-apps/plugin-sql", () => ({
  default: {
    load: loadMock,
  },
}));

import {
  BACKUP_DIRECTORY,
  DATABASE_DIRECTORY,
  DATABASE_FILE_NAME,
  DATABASE_PATH,
  backupTrajonDatabase,
  closeTrajonDatabase,
  openTrajonDatabase,
  restoreTrajonDatabase,
} from "../sqliteStorage";

declare global {
  interface Window {
    __TAURI_INTERNALS__?: unknown;
  }
}

describe("sqliteStorage", () => {
  beforeEach(() => {
    delete window.__TAURI_INTERNALS__;
    vi.mocked(copyFile).mockReset();
    vi.mocked(exists).mockReset();
    vi.mocked(mkdir).mockReset();
    vi.mocked(remove).mockReset();
    closeMock.mockReset();
    loadMock.mockReset();
    loadMock.mockResolvedValue({ close: closeMock });
  });

  it("opens the app-scoped SQLite database in Tauri", async () => {
    window.__TAURI_INTERNALS__ = {};

    const db = await openTrajonDatabase();

    expect(mkdir).toHaveBeenCalledWith(DATABASE_DIRECTORY, {
      baseDir: BaseDirectory.AppData,
      recursive: true,
    });
    expect(Database.load).toHaveBeenCalledWith(DATABASE_PATH);
    expect(db).toEqual({ close: closeMock });
  });

  it("rejects desktop database access outside Tauri", async () => {
    await expect(openTrajonDatabase()).rejects.toThrow(
      "Tauri SQLite storage is only available in the desktop app",
    );
  });

  it("backs up the SQLite file into the app backup directory", async () => {
    window.__TAURI_INTERNALS__ = {};
    vi.mocked(exists).mockResolvedValue(true);

    await openTrajonDatabase();
    const backupPath = await backupTrajonDatabase("backup.db");

    expect(closeMock).toHaveBeenCalledTimes(1);
    expect(mkdir).toHaveBeenCalledWith(BACKUP_DIRECTORY, {
      baseDir: BaseDirectory.AppData,
      recursive: true,
    });
    expect(copyFile).toHaveBeenCalledWith(
      `${DATABASE_DIRECTORY}/${DATABASE_FILE_NAME}`,
      `${BACKUP_DIRECTORY}/backup.db`,
      {
        fromPathBaseDir: BaseDirectory.AppData,
        toPathBaseDir: BaseDirectory.AppData,
      },
    );
    expect(backupPath).toBe(`${BACKUP_DIRECTORY}/backup.db`);
  });

  it("rejects backup when the database file does not exist yet", async () => {
    window.__TAURI_INTERNALS__ = {};
    vi.mocked(exists).mockResolvedValue(false);

    await expect(backupTrajonDatabase("backup.db")).rejects.toThrow(
      "SQLite database file does not exist yet",
    );
  });

  it("restores the SQLite file from a backup in the app directory", async () => {
    window.__TAURI_INTERNALS__ = {};
    vi.mocked(exists)
      .mockResolvedValueOnce(true)
      .mockResolvedValueOnce(true);

    await openTrajonDatabase();
    await restoreTrajonDatabase("backup.db");

    expect(closeMock).toHaveBeenCalledTimes(1);
    expect(remove).toHaveBeenCalledWith(`${DATABASE_DIRECTORY}/${DATABASE_FILE_NAME}`, {
      baseDir: BaseDirectory.AppData,
    });
    expect(copyFile).toHaveBeenCalledWith(
      `${BACKUP_DIRECTORY}/backup.db`,
      `${DATABASE_DIRECTORY}/${DATABASE_FILE_NAME}`,
      {
        fromPathBaseDir: BaseDirectory.AppData,
        toPathBaseDir: BaseDirectory.AppData,
      },
    );
  });

  it("rejects restore when the backup file does not exist", async () => {
    window.__TAURI_INTERNALS__ = {};
    vi.mocked(exists).mockResolvedValue(false);

    await expect(restoreTrajonDatabase("missing.db")).rejects.toThrow(
      "SQLite backup file does not exist",
    );
  });

  it("closes the cached database connection once", async () => {
    window.__TAURI_INTERNALS__ = {};

    await openTrajonDatabase();
    await closeTrajonDatabase();
    await closeTrajonDatabase();

    expect(closeMock).toHaveBeenCalledTimes(1);
  });
});
