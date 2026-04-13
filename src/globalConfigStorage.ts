import { BaseDirectory, exists, mkdir, readTextFile, writeTextFile } from "@tauri-apps/plugin-fs";

import type { GlobalConfig } from "./model/globalConfig";
import type { TradingPanelState } from "./model/storage";
import {
  TIMEFRAME_PRESET_VALUES,
  type TimeframePreset,
  type TradingPairPreset,
} from "./model/tradingPairPreset";

export const GLOBAL_CONFIG_DIRECTORY = ".trading-flow";
export const GLOBAL_CONFIG_FILE_NAME = "config.json";
export const GLOBAL_CONFIG_FILE_PATH = `${GLOBAL_CONFIG_DIRECTORY}/${GLOBAL_CONFIG_FILE_NAME}`;
export const GLOBAL_CONFIG_FALLBACK_STORAGE_KEY = "trading-flow:global-config";

const DEFAULT_CONFIG: GlobalConfig = {
  timerEnabled: false,
  timerInterval: 300,
};

const TIMEFRAME_PRESETS = new Set<string>(TIMEFRAME_PRESET_VALUES);

export type StoredGlobalConfig = Pick<
  TradingPanelState,
  "config" | "pairPresets" | "defaultPresetSymbol"
>;

function isTauriRuntime(): boolean {
  return typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function canUseWindowStorage(): boolean {
  return typeof window !== "undefined" && typeof window.localStorage !== "undefined";
}

export function readGlobalConfigFallback(): StoredGlobalConfig | null {
  if (!canUseWindowStorage()) {
    return null;
  }

  const raw = window.localStorage.getItem(GLOBAL_CONFIG_FALLBACK_STORAGE_KEY);
  if (!raw) {
    return null;
  }

  try {
    return normalizeStoredGlobalConfig(JSON.parse(raw));
  } catch {
    return null;
  }
}

function normalizeSymbol(value: string): string {
  return value.trim().toUpperCase();
}

function normalizeConfig(value: unknown): GlobalConfig {
  if (!isRecord(value)) {
    return { ...DEFAULT_CONFIG };
  }

   const timerInterval =
    typeof value.timerInterval === "number" &&
    Number.isFinite(value.timerInterval) &&
    value.timerInterval > 0 &&
    Number.isInteger(value.timerInterval)
      ? value.timerInterval
      : DEFAULT_CONFIG.timerInterval;

  return {
    timerEnabled:
      typeof value.timerEnabled === "boolean"
        ? value.timerEnabled
        : DEFAULT_CONFIG.timerEnabled,
    timerInterval,
  };
}

function normalizePairPreset(value: unknown): TradingPairPreset | undefined {
  if (!isRecord(value) || typeof value.symbol !== "string") {
    return undefined;
  }

  const symbol = normalizeSymbol(value.symbol);
  if (!symbol || typeof value.timeframe !== "string" || !TIMEFRAME_PRESETS.has(value.timeframe)) {
    return undefined;
  }

  return {
    symbol,
    timeframe: value.timeframe as TimeframePreset,
  };
}

function normalizePairPresets(value: unknown): TradingPairPreset[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value.flatMap((item) => {
    const preset = normalizePairPreset(item);
    return preset ? [preset] : [];
  });
}

function normalizeDefaultPresetSymbol(
  defaultPresetSymbol: unknown,
  pairPresets: TradingPairPreset[],
): string | undefined {
  if (typeof defaultPresetSymbol !== "string") {
    return undefined;
  }

  const normalizedSymbol = normalizeSymbol(defaultPresetSymbol);
  if (!normalizedSymbol) {
    return undefined;
  }

  return pairPresets.some((preset) => preset.symbol === normalizedSymbol)
    ? normalizedSymbol
    : undefined;
}

function normalizeStoredGlobalConfig(value: unknown): StoredGlobalConfig {
  const record = isRecord(value) ? value : {};
  const pairPresets = normalizePairPresets(record.pairPresets);

  return {
    config: normalizeConfig(record.config),
    pairPresets,
    defaultPresetSymbol: normalizeDefaultPresetSymbol(record.defaultPresetSymbol, pairPresets),
  };
}

async function ensureGlobalConfigDirectory(): Promise<void> {
  await mkdir(GLOBAL_CONFIG_DIRECTORY, {
    baseDir: BaseDirectory.Home,
    recursive: true,
  });
}

export async function readGlobalConfig(): Promise<StoredGlobalConfig | null> {
  if (!isTauriRuntime()) {
    return readGlobalConfigFallback();
  }

  const fileExists = await exists(GLOBAL_CONFIG_FILE_PATH, {
    baseDir: BaseDirectory.Home,
  });

  if (!fileExists) {
    return null;
  }

  try {
    const contents = await readTextFile(GLOBAL_CONFIG_FILE_PATH, {
      baseDir: BaseDirectory.Home,
    });

    return normalizeStoredGlobalConfig(JSON.parse(contents));
  } catch (error) {
    if (error instanceof Error && /no such file|not found/i.test(error.message)) {
      return null;
    }

    if (error instanceof SyntaxError) {
      return null;
    }

    throw error;
  }
}

export async function writeGlobalConfig(value: StoredGlobalConfig): Promise<void> {
  if (!isTauriRuntime()) {
    if (canUseWindowStorage()) {
      window.localStorage.setItem(
        GLOBAL_CONFIG_FALLBACK_STORAGE_KEY,
        JSON.stringify(normalizeStoredGlobalConfig(value)),
      );
    }

    return;
  }

  await ensureGlobalConfigDirectory();
  await writeTextFile(GLOBAL_CONFIG_FILE_PATH, JSON.stringify(normalizeStoredGlobalConfig(value)), {
    baseDir: BaseDirectory.Home,
  });
}
