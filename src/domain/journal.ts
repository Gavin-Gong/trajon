import type {
  TradeEvent,
  TradeEventType,
  TradeJournal,
  TradeJournalStatus,
} from "../model/model";

import { isZero } from "./shared";

export type {
  TradeEvent,
  TradeEventType,
  TradeJournal,
  TradeJournalStatus,
} from "../model/model";

export type TradeEventDraft = {
  type: TradeEventType;
  content: string;
  priceInput: string;
  quantityInput: string;
};

export type TradeEventComposerError =
  | "missingContent"
  | "missingQuantity"
  | "invalidPrecision"
  | "zeroQuantity"
  | "wrongDirection"
  | "exceedsPosition"
  | "doneOnlyLogs";

export type TradeEventComposerResult =
  | { ok: true; event: TradeEvent }
  | { ok: false; error: TradeEventComposerError };

type QuantityValidationResult =
  | { ok: true; quantity: number }
  | { ok: false; error: TradeEventComposerError };

const DECIMAL_PRECISION_LIMIT = 2;

export function createJournal(
  overrides: Partial<TradeJournal> = {},
): TradeJournal {
  const createdAt = overrides.createdAt ?? new Date().toISOString();

  return {
    id: overrides.id ?? createId(),
    createdAt,
    updatedAt: overrides.updatedAt ?? createdAt,
    symbol: overrides.symbol ?? "",
    market: overrides.market ?? "",
    summary: overrides.summary ?? "",
    status: overrides.status ?? "open",
    events: overrides.events ?? [],
  };
}

export function netPosition(events: TradeEvent[]): number {
  return events.reduce((total, event) => total + (event.quantity ?? 0), 0);
}

export function availableEventTypes(
  status: TradeJournalStatus,
  currentNetPosition: number,
): TradeEventType[] {
  if (status === "done") {
    return ["log"];
  }

  return isZero(currentNetPosition)
    ? ["log", "entry"]
    : ["log", "add", "reduce", "exit"];
}

export function currentJournal(
  journals: TradeJournal[],
): TradeJournal | undefined {
  return [...journals]
    .filter((journal) => journal.status === "open")
    .sort(compareJournalForPanel)[0];
}

export function composeTradeEvent(
  draft: TradeEventDraft,
  currentNetPosition: number,
  status: TradeJournalStatus,
): TradeEventComposerResult {
  const trimmedContent = draft.content.trim();
  const price = parseOptionalDecimal(draft.priceInput);

  if (status === "done" && draft.type !== "log") {
    return { ok: false, error: "doneOnlyLogs" };
  }

  if (draft.type === "log") {
    if (!trimmedContent) {
      return { ok: false, error: "missingContent" };
    }

    return {
      ok: true,
      event: buildEvent({
        type: "log",
        content: trimmedContent,
        price,
      }),
    };
  }

  if (draft.type === "exit") {
    if (isZero(currentNetPosition)) {
      return { ok: false, error: "zeroQuantity" };
    }

    return {
      ok: true,
      event: buildEvent({
        type: "exit",
        content: trimmedContent,
        price,
        quantity: -currentNetPosition,
      }),
    };
  }

  const quantityValidation = validateQuantity(
    draft.quantityInput,
    draft.type,
    currentNetPosition,
  );
  if (!quantityValidation.ok) {
    return quantityValidation;
  }

  const normalizedType =
    draft.type === "reduce" &&
    shouldConvertReduceToExit(quantityValidation.quantity, currentNetPosition)
      ? "exit"
      : draft.type;

  return {
    ok: true,
    event: buildEvent({
      type: normalizedType,
      content: trimmedContent,
      price,
      quantity:
        normalizedType === "exit"
          ? -currentNetPosition
          : quantityValidation.quantity,
    }),
  };
}

function buildEvent(
  input: Pick<TradeEvent, "type" | "content"> & {
    price?: number;
    quantity?: number;
  },
): TradeEvent {
  const timestamp = new Date().toISOString();

  return {
    id: createId(),
    createdAt: timestamp,
    updatedAt: timestamp,
    timestamp,
    type: input.type,
    content: input.content,
    price: input.price,
    quantity: input.quantity,
  };
}

function validateQuantity(
  quantityInput: string,
  type: Exclude<TradeEventType, "log" | "exit">,
  currentNetPosition: number,
): QuantityValidationResult {
  const trimmed = quantityInput.trim();
  if (!trimmed) {
    return { ok: false, error: "missingQuantity" };
  }

  if (!hasAtMostTwoDecimals(trimmed)) {
    return { ok: false, error: "invalidPrecision" };
  }

  const quantity = Number(trimmed);
  if (Number.isNaN(quantity)) {
    return { ok: false, error: "missingQuantity" };
  }

  if (isZero(quantity)) {
    return { ok: false, error: "zeroQuantity" };
  }

  if (type === "add" && !isSameDirection(quantity, currentNetPosition)) {
    return { ok: false, error: "wrongDirection" };
  }

  if (type === "reduce") {
    if (!isOppositeDirection(quantity, currentNetPosition)) {
      return { ok: false, error: "wrongDirection" };
    }

    if (Math.abs(quantity) > Math.abs(currentNetPosition)) {
      return { ok: false, error: "exceedsPosition" };
    }
  }

  return { ok: true, quantity };
}

function hasAtMostTwoDecimals(value: string): boolean {
  const [whole, fraction, ...rest] = value.split(".");
  if (rest.length > 0) {
    return false;
  }

  if (whole.length === 0 && fraction === undefined) {
    return false;
  }

  return fraction === undefined || fraction.length <= DECIMAL_PRECISION_LIMIT;
}

function parseOptionalDecimal(value: string): number | undefined {
  const trimmed = value.trim();
  if (!trimmed) {
    return undefined;
  }

  if (!hasAtMostTwoDecimals(trimmed)) {
    return undefined;
  }

  const parsed = Number(trimmed);
  return Number.isNaN(parsed) ? undefined : parsed;
}

function shouldConvertReduceToExit(
  quantity: number,
  currentNetPosition: number,
): boolean {
  return !isZero(currentNetPosition) && isZero(currentNetPosition + quantity);
}

function isSameDirection(quantity: number, currentNetPosition: number): boolean {
  return !isZero(quantity) && !isZero(currentNetPosition)
    ? Math.sign(quantity) === Math.sign(currentNetPosition)
    : false;
}

function isOppositeDirection(
  quantity: number,
  currentNetPosition: number,
): boolean {
  return !isZero(quantity) && !isZero(currentNetPosition)
    ? Math.sign(quantity) !== Math.sign(currentNetPosition)
    : false;
}

function compareJournalForPanel(left: TradeJournal, right: TradeJournal): number {
  if (left.status !== right.status) {
    return left.status === "open" ? -1 : 1;
  }

  const symbolOrder = left.symbol.toLowerCase().localeCompare(right.symbol.toLowerCase());
  if (symbolOrder !== 0) {
    return symbolOrder;
  }

  return left.id.localeCompare(right.id);
}

function createId(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }

  return `id-${Math.random().toString(36).slice(2, 10)}`;
}
