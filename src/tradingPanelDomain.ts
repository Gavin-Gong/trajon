import type {
  TradeEvent,
  TradeEventType,
  TradeJournal,
  TradeJournalStatus,
} from "./model/model";

export type {
  TradeEvent,
  TradeEventType,
  TradeJournal,
  TradeJournalStatus,
} from "./model/model";

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

export type CalendarDay = {
  date: Date;
  isInDisplayedMonth: boolean;
};

export type CalendarDayGroup = {
  date: Date;
  dayKey: string;
  journals: TradeJournal[];
};

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

export function groupJournalsByDay(
  journals: TradeJournal[],
): CalendarDayGroup[] {
  const groups = new Map<string, TradeJournal[]>();

  for (const journal of journals) {
    const dayKey = toUtcDayKey(journal.createdAt);
    const existing = groups.get(dayKey) ?? [];
    existing.push(journal);
    groups.set(dayKey, existing);
  }

  return [...groups.entries()]
    .sort(([leftKey], [rightKey]) => leftKey.localeCompare(rightKey))
    .map(([dayKey, dayJournals]) => ({
      dayKey,
      date: new Date(`${dayKey}T00:00:00.000Z`),
      journals: [...dayJournals].sort(compareJournalForCalendar),
    }));
}

export function defaultCalendarSelection(
  displayedMonth: Date,
  journals: TradeJournal[],
): TradeJournal | undefined {
  const { start, end } = utcMonthBounds(displayedMonth);

  return [...journals]
    .filter((journal) => {
      const createdTime = new Date(journal.createdAt).getTime();
      return createdTime >= start.getTime() && createdTime < end.getTime();
    })
    .sort(compareJournalForCalendar)[0];
}

export function visibleMonthDays(displayedMonth: Date): CalendarDay[] {
  const { start, end } = utcMonthBounds(displayedMonth);
  const monthEnd = new Date(end.getTime() - ONE_DAY_MS);

  const leadingCount = start.getUTCDay();
  const gridStart = new Date(start.getTime() - leadingCount * ONE_DAY_MS);
  const trailingCount = (7 - ((leadingCount + monthEnd.getUTCDate()) % 7)) % 7;
  const visibleCount = leadingCount + monthEnd.getUTCDate() + trailingCount;

  return Array.from({ length: visibleCount }, (_, offset) => {
    const date = new Date(gridStart.getTime() + offset * ONE_DAY_MS);

    return {
      date,
      isInDisplayedMonth:
        date.getUTCFullYear() === start.getUTCFullYear() &&
        date.getUTCMonth() === start.getUTCMonth(),
    } satisfies CalendarDay;
  });
}

export function isZero(value: number, tolerance = 0.000001): boolean {
  return Math.abs(value) < tolerance;
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

function compareJournalForCalendar(
  left: TradeJournal,
  right: TradeJournal,
): number {
  const timeOrder = new Date(left.createdAt).getTime() - new Date(right.createdAt).getTime();
  if (timeOrder !== 0) {
    return timeOrder;
  }

  const symbolOrder = left.symbol.toLowerCase().localeCompare(right.symbol.toLowerCase());
  if (symbolOrder !== 0) {
    return symbolOrder;
  }

  return left.id.localeCompare(right.id);
}

function utcMonthBounds(date: Date): { start: Date; end: Date } {
  const start = new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), 1));
  const end = new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth() + 1, 1));
  return { start, end };
}

function toUtcDayKey(value: string): string {
  const date = new Date(value);
  const year = date.getUTCFullYear();
  const month = `${date.getUTCMonth() + 1}`.padStart(2, "0");
  const day = `${date.getUTCDate()}`.padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function createId(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }

  return `id-${Math.random().toString(36).slice(2, 10)}`;
}

const ONE_DAY_MS = 24 * 60 * 60 * 1000;
