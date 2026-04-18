import { BaseDirectory, copyFile, exists, mkdir, remove } from "@tauri-apps/plugin-fs";
import Database from "@tauri-apps/plugin-sql";

export const DATABASE_DIRECTORY = "databases";
export const BACKUP_DIRECTORY = `${DATABASE_DIRECTORY}/backups`;
export const DATABASE_FILE_NAME = "trajon.db";
export const DATABASE_PATH = `sqlite:${DATABASE_DIRECTORY}/${DATABASE_FILE_NAME}`;

const SQLITE_UNAVAILABLE_ERROR = "Tauri SQLite storage is only available in the desktop app";

let cachedDatabase: Database | null = null;

function isTauriRuntime(): boolean {
  return typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;
}

function assertTauriSqlAvailable(): void {
  if (!isTauriRuntime()) {
    throw new Error(SQLITE_UNAVAILABLE_ERROR);
  }
}

async function ensureDatabaseDirectory(): Promise<void> {
  await mkdir(DATABASE_DIRECTORY, {
    baseDir: BaseDirectory.AppData,
    recursive: true,
  });
}

async function ensureBackupDirectory(): Promise<void> {
  await mkdir(BACKUP_DIRECTORY, {
    baseDir: BaseDirectory.AppData,
    recursive: true,
  });
}

function getDatabaseFilePath(): string {
  return `${DATABASE_DIRECTORY}/${DATABASE_FILE_NAME}`;
}

export async function openTrajonDatabase(): Promise<Database> {
  assertTauriSqlAvailable();

  if (cachedDatabase) {
    return cachedDatabase;
  }

  await ensureDatabaseDirectory();
  cachedDatabase = await Database.load(DATABASE_PATH);
  return cachedDatabase;
}

export async function closeTrajonDatabase(): Promise<void> {
  if (!cachedDatabase) {
    return;
  }

  const db = cachedDatabase;
  cachedDatabase = null;
  await db.close();
}

export async function backupTrajonDatabase(fileName: string): Promise<string> {
  assertTauriSqlAvailable();

  const databaseFilePath = getDatabaseFilePath();
  const backupPath = `${BACKUP_DIRECTORY}/${fileName.trim()}`;

  if (!(await exists(databaseFilePath, { baseDir: BaseDirectory.AppData }))) {
    throw new Error("SQLite database file does not exist yet");
  }

  await closeTrajonDatabase();
  await ensureBackupDirectory();
  await copyFile(databaseFilePath, backupPath, {
    fromPathBaseDir: BaseDirectory.AppData,
    toPathBaseDir: BaseDirectory.AppData,
  });
  return backupPath;
}

export async function restoreTrajonDatabase(fileName: string): Promise<void> {
  assertTauriSqlAvailable();

  const backupPath = `${BACKUP_DIRECTORY}/${fileName.trim()}`;
  if (!(await exists(backupPath, { baseDir: BaseDirectory.AppData }))) {
    throw new Error("SQLite backup file does not exist");
  }

  await closeTrajonDatabase();
  await ensureDatabaseDirectory();

  const databaseFilePath = getDatabaseFilePath();
  if (await exists(databaseFilePath, { baseDir: BaseDirectory.AppData })) {
    await remove(databaseFilePath, { baseDir: BaseDirectory.AppData });
  }

  await copyFile(backupPath, databaseFilePath, {
    fromPathBaseDir: BaseDirectory.AppData,
    toPathBaseDir: BaseDirectory.AppData,
  });
}
