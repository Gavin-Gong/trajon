import { BaseDirectory, exists, mkdir, readTextFile, writeTextFile } from "@tauri-apps/plugin-fs";

export const DEMO_DIRECTORY = ".trading-flow";
export const DEMO_FILE_NAME = "demo.json";
export const DEMO_FILE_PATH = `${DEMO_DIRECTORY}/${DEMO_FILE_NAME}`;

const TAURI_FS_UNAVAILABLE_ERROR = "Tauri file storage is only available in the desktop app";
const DEMO_NULL_VALUE_ERROR = "Demo JSON value cannot be null";

function isTauriRuntime(): boolean {
  return typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;
}

function assertTauriFsAvailable(): void {
  if (!isTauriRuntime()) {
    throw new Error(TAURI_FS_UNAVAILABLE_ERROR);
  }
}

async function ensureDemoDirectory(): Promise<void> {
  await mkdir(DEMO_DIRECTORY, {
    baseDir: BaseDirectory.Home,
    recursive: true,
  });
}

export async function readDemoJson(): Promise<unknown> {
  assertTauriFsAvailable();

  const fileExists = await exists(DEMO_FILE_PATH, {
    baseDir: BaseDirectory.Home,
  });

  if (!fileExists) {
    return null;
  }

  let contents: string;

  try {
    contents = await readTextFile(DEMO_FILE_PATH, {
      baseDir: BaseDirectory.Home,
    });
  } catch (error) {
    if (error instanceof Error && /no such file|not found/i.test(error.message)) {
      return null;
    }

    throw error;
  }

  try {
    return JSON.parse(contents);
  } catch {
    throw new Error(`Invalid JSON in ${DEMO_FILE_NAME}`);
  }
}

export async function writeDemoJson(value: unknown): Promise<void> {
  assertTauriFsAvailable();

  if (value === null) {
    throw new Error(DEMO_NULL_VALUE_ERROR);
  }

  await ensureDemoDirectory();
  await writeTextFile(DEMO_FILE_PATH, JSON.stringify(value), {
    baseDir: BaseDirectory.Home,
  });
}
