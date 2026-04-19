import type { TradeJournal } from "./journal";
import { toUtcDayKey } from "./shared";

export type CalendarDay = {
  date: Date;
  isInDisplayedMonth: boolean;
};

export type CalendarDayGroup = {
  date: Date;
  dayKey: string;
  journals: TradeJournal[];
};

const ONE_DAY_MS = 24 * 60 * 60 * 1000;

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
