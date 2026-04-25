import { convertFileSrc } from "@tauri-apps/api/core";
import { homeDir, join, normalize } from "@tauri-apps/api/path";
import { exists, mkdir, remove, writeFile } from "@tauri-apps/plugin-fs";

import type { JournalImage, TradeJournal } from "../domain/journal";

export const JOURNAL_IMAGE_ROOT_DIRECTORY = "trajon";
export const JOURNAL_IMAGE_DIRECTORY = `${JOURNAL_IMAGE_ROOT_DIRECTORY}/images`;

export type JournalImageInput = File | string;

export function isTauriRuntime(): boolean {
  return typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;
}

export function isJournalImageManagementAvailable(): boolean {
  return isTauriRuntime() && typeof File !== "undefined";
}

export async function pickJournalImageFiles(): Promise<File[]> {
  if (!isJournalImageManagementAvailable()) {
    return [];
  }

  return new Promise((resolve) => {
    const input = document.createElement("input");
    input.type = "file";
    input.accept = "image/*";
    input.multiple = true;
    input.onchange = () => resolve(Array.from(input.files ?? []));
    input.click();
  });
}

export async function persistJournalImages(
  journal: TradeJournal,
  files: JournalImageInput[],
): Promise<JournalImage[]> {
  if (!isTauriRuntime()) {
    return [];
  }

  const imagesDirectory = await ensureJournalImagesDirectory();
  const existingCount = journal.images.length;

  return Promise.all(
    files.map(async (file, index) => {
      const timestamp = Date.now() + index;
      const extension = getImageExtension(file);
      const fileName = buildJournalImageFileName(journal.id, timestamp, journal.symbol, extension);
      const destinationPath = await join(imagesDirectory, fileName);

      if (typeof file !== "string") {
        const bytes = new Uint8Array(await file.arrayBuffer());
        await writeFile(destinationPath, bytes);
      }

      return {
        id: crypto.randomUUID(),
        journalId: journal.id,
        createdAt: new Date(timestamp).toISOString(),
        updatedAt: new Date(timestamp).toISOString(),
        fileName,
        relativePath: `${JOURNAL_IMAGE_DIRECTORY}/${fileName}`,
        sortOrder: existingCount + index,
      } satisfies JournalImage;
    }),
  );
}

export async function removeJournalImageFile(relativePath: string): Promise<void> {
  if (!isTauriRuntime()) {
    return;
  }

  const absolutePath = await resolveJournalImageAbsolutePath(relativePath);
  if (!(await exists(absolutePath))) {
    return;
  }

  await remove(absolutePath);
}

export async function resolveJournalImageSrc(relativePath: string): Promise<string> {
  if (!isTauriRuntime()) {
    return relativePath;
  }

  return convertFileSrc(await resolveJournalImageAbsolutePath(relativePath));
}

export function buildJournalImageFileName(
  journalId: string,
  timestamp: number,
  symbol: string,
  extension: string,
): string {
  const normalizedSymbol = symbol.trim().toLowerCase().replace(/[^a-z0-9_-]+/g, "-") || "unknown";
  const normalizedExtension = extension.startsWith(".") ? extension : `.${extension}`;
  return `${journalId}-${timestamp}-${normalizedSymbol}${normalizedExtension}`;
}

async function ensureJournalImagesDirectory(): Promise<string> {
  const home = await homeDir();
  const root = await join(home, JOURNAL_IMAGE_ROOT_DIRECTORY);
  const imagesDirectory = await join(root, "images");
  await mkdir(imagesDirectory, { recursive: true });
  return imagesDirectory;
}

async function resolveJournalImageAbsolutePath(relativePath: string): Promise<string> {
  const home = await homeDir();
  return normalize(await join(home, relativePath));
}

function getImageExtension(file: JournalImageInput): string {
  const source = typeof file === "string" ? file : file.name;
  const match = source.match(/(\.[a-zA-Z0-9]+)$/);
  return match ? match[1] : ".png";
}
