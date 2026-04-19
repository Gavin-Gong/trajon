import { z } from "zod";

import { TIMEFRAME_PRESET_VALUES } from "../model/tradingPairPreset";

const isoDateTimeSchema = z.iso.datetime();
const trimmedStringSchema = z.string().trim();
const optionalTrimmedStringSchema = z.string().trim().optional();
const normalizedSymbolSchema = z.string().trim().min(1).transform((value) => value.toUpperCase());

export const tradeJournalStatusSchema = z.enum(["open", "done"]);
export const tradeEventTypeSchema = z.enum(["entry", "add", "reduce", "exit", "log"]);

export const tradeEventSchema = z.object({
  id: z.string().min(1),
  createdAt: isoDateTimeSchema,
  updatedAt: isoDateTimeSchema,
  timestamp: isoDateTimeSchema,
  type: tradeEventTypeSchema,
  content: trimmedStringSchema,
  price: z.number().finite().optional(),
  quantity: z.number().finite().optional(),
});

export const tradeJournalSchema = z.object({
  id: z.string().min(1),
  createdAt: isoDateTimeSchema,
  updatedAt: isoDateTimeSchema,
  symbol: normalizedSymbolSchema,
  market: trimmedStringSchema,
  summary: z.string(),
  status: tradeJournalStatusSchema,
  events: z.array(tradeEventSchema),
});

export const tradeJournalRowSchema = z.object({
  id: z.string().min(1),
  created_at: isoDateTimeSchema,
  updated_at: isoDateTimeSchema,
  symbol: normalizedSymbolSchema,
  market: trimmedStringSchema,
  summary: z.string(),
  status: tradeJournalStatusSchema,
});

export const tradeEventRowSchema = z.object({
  id: z.string().min(1),
  journal_id: z.string().min(1),
  created_at: isoDateTimeSchema,
  updated_at: isoDateTimeSchema,
  timestamp: isoDateTimeSchema,
  type: tradeEventTypeSchema,
  content: trimmedStringSchema,
  price: z.number().finite().nullable().optional(),
  quantity: z.number().finite().nullable().optional(),
});

export const globalConfigSchema = z.object({
  timerEnabled: z.boolean(),
  timerInterval: z.number().int().positive(),
});

export const tradingPairPresetSchema = z.object({
  symbol: normalizedSymbolSchema,
  timeframe: z.enum(TIMEFRAME_PRESET_VALUES),
});

export const tradingJournalStateSchema = z.object({
  journals: z.array(tradeJournalSchema),
  config: globalConfigSchema,
  pairPresets: z.array(tradingPairPresetSchema),
  defaultPresetSymbol: optionalTrimmedStringSchema.transform((value) => value?.toUpperCase()),
  panelPinned: z.boolean(),
});

export const appConfigValueSchema = z.object({
  config: globalConfigSchema.optional(),
  pairPresets: z.array(tradingPairPresetSchema).optional(),
  defaultPresetSymbol: optionalTrimmedStringSchema.transform((value) => value?.toUpperCase()),
  panelPinned: z.boolean().optional(),
});
