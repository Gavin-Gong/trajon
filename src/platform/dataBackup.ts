import { homeDir, join } from "@tauri-apps/api/path";
import { readDir } from "@tauri-apps/plugin-fs";

import { BACKUP_DIRECTORY, backupTrajonDatabase, restoreTrajonDatabase } from "../database/sqliteStorage";

export type DatabaseBackupFile = {
  fileName: string;
  absolutePath: string;
};

type DatabaseBackupResult = {
  fileName: string;
  relativePath: string;
};

export function isDatabaseBackupAvailable(): boolean {
  return typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;
}

export function buildDatabaseBackupFileName(now: Date = new Date()): string {
  const year = String(now.getUTCFullYear());
  const month = String(now.getUTCMonth() + 1).padStart(2, "0");
  const day = String(now.getUTCDate()).padStart(2, "0");
  const hours = String(now.getUTCHours()).padStart(2, "0");
  const minutes = String(now.getUTCMinutes()).padStart(2, "0");
  const seconds = String(now.getUTCSeconds()).padStart(2, "0");
  return `trajon-backup-${year}${month}${day}-${hours}${minutes}${seconds}.db`;
}

export function normalizeDatabaseBackupFileName(fileName: string): string {
  const trimmed = fileName.trim();
  if (!trimmed) {
    return "";
  }

  return trimmed.toLowerCase().endsWith(".db") ? trimmed : `${trimmed}.db`;
}

export async function createDatabaseBackup(fileName?: string): Promise<DatabaseBackupResult> {
  const nextFileName = normalizeDatabaseBackupFileName(fileName ?? buildDatabaseBackupFileName());
  if (!nextFileName) {
    throw new Error("请输入备份文件名");
  }

  const relativePath = await backupTrajonDatabase(nextFileName);
  return { fileName: nextFileName, relativePath };
}

export async function restoreDatabaseBackup(fileName: string): Promise<void> {
  await restoreTrajonDatabase(fileName);
}

export async function listDatabaseBackups(): Promise<DatabaseBackupFile[]> {
  const root = await homeDir();
  const backupDirectory = await join(root, BACKUP_DIRECTORY);
  const entries = await readDir(backupDirectory);

  return entries
    .filter((entry) => entry.isFile && entry.name?.toLowerCase().endsWith(".db"))
    .map((entry) => ({
      fileName: entry.name!,
      absolutePath: `${backupDirectory}/${entry.name!}`,
    }))
    .sort((left, right) => right.fileName.localeCompare(left.fileName));
}
