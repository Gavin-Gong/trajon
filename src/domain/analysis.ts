import type { TradeJournal } from "./journal";
import { isZero, roundToTwo, toUtcDayKey } from "./shared";

const HEATMAP_BUCKETS = 4;
const HEATMAP_DAY_COUNT = 53 * 7;

export type RealizedPnlPoint = {
  timestamp: string;
  value: number;
};

export type DailyRealizedPnlPoint = {
  dayKey: string;
  value: number;
};

export type DailyRealizedPnlHeatmapCell = {
  dayKey: string;
  value: number;
  bucket: number;
  direction: "profit" | "loss" | "flat";
};

export type RealizedPnlAnalytics = {
  realizedTotal: number;
  equityCurve: RealizedPnlPoint[];
  dailyPnl: DailyRealizedPnlPoint[];
  heatmap: DailyRealizedPnlHeatmapCell[];
  profitDays: number;
  lossDays: number;
  journalCount: number;
};

export type StrategyAnalytics = {
  todayRealizedPnl?: number;
  todayTradeCount: number;
  todayWinningJournalCount: number;
  todayLosingJournalCount: number;
  todayWinningPnl?: number;
  winRate?: number;
  winningJournalCount: number;
  completedJournalCount: number;
  profitLossRatio?: number;
  averageWin?: number;
  averageLoss?: number;
};

type RealizedPnlAnalyticsOptions = {
  heatmapEndDayKey?: string;
};

export function realizedPnlAnalytics(
  journals: TradeJournal[],
  options: RealizedPnlAnalyticsOptions = {},
): RealizedPnlAnalytics {
  const completedJournals = journals.filter((journal) => journal.status === "done");
  const realizedEvents: Array<{ timestamp: string; pnl: number }> = [];

  for (const journal of completedJournals) {
    realizedEvents.push(...realizedEventsForJournal(journal));
  }

  realizedEvents.sort(
    (left, right) => new Date(left.timestamp).getTime() - new Date(right.timestamp).getTime(),
  );

  let realizedTotal = 0;
  const equityCurve = realizedEvents.map((event) => {
    realizedTotal = roundToTwo(realizedTotal + event.pnl);
    return { timestamp: event.timestamp, value: realizedTotal } satisfies RealizedPnlPoint;
  });

  const dailyMap = new Map<string, number>();
  for (const event of realizedEvents) {
    const dayKey = toUtcDayKey(event.timestamp);
    dailyMap.set(dayKey, roundToTwo((dailyMap.get(dayKey) ?? 0) + event.pnl));
  }

  const dailyPnl = [...dailyMap.entries()]
    .sort(([leftKey], [rightKey]) => leftKey.localeCompare(rightKey))
    .map(([dayKey, value]) => ({ dayKey, value }) satisfies DailyRealizedPnlPoint);
  const heatmap = buildHeatmap(dailyMap, options.heatmapEndDayKey);

  return {
    realizedTotal,
    equityCurve,
    dailyPnl,
    heatmap,
    profitDays: dailyPnl.filter((item) => item.value > 0).length,
    lossDays: dailyPnl.filter((item) => item.value < 0).length,
    journalCount: completedJournals.length,
  };
}

export function strategyAnalytics(
  journals: TradeJournal[],
  options: RealizedPnlAnalyticsOptions = {},
): StrategyAnalytics {
  const completedJournals = journals.filter((journal) => journal.status === "done");
  const todayDayKey = options.heatmapEndDayKey ?? toUtcDayKey(new Date().toISOString());
  const realizedByJournal = completedJournals.map((journal) => ({
    journal,
    realizedPnl: roundToTwo(
      realizedEventsForJournal(journal).reduce((sum, event) => roundToTwo(sum + event.pnl), 0),
    ),
  }));
  const todayJournals = realizedByJournal.filter(
    ({ journal }) => toUtcDayKey(journal.updatedAt) === todayDayKey,
  );
  const todayWins = todayJournals.filter(({ realizedPnl }) => realizedPnl > 0);
  const todayLosses = todayJournals.filter(({ realizedPnl }) => realizedPnl < 0);
  const wins = realizedByJournal.filter(({ realizedPnl }) => realizedPnl > 0);
  const losses = realizedByJournal.filter(({ realizedPnl }) => realizedPnl < 0);

  return {
    todayRealizedPnl:
      todayJournals.length > 0
        ? roundToTwo(todayJournals.reduce((sum, item) => roundToTwo(sum + item.realizedPnl), 0))
        : undefined,
    todayTradeCount: todayJournals.length,
    todayWinningJournalCount: todayWins.length,
    todayLosingJournalCount: todayLosses.length,
    todayWinningPnl:
      todayWins.length > 0
        ? roundToTwo(todayWins.reduce((sum, item) => roundToTwo(sum + item.realizedPnl), 0))
        : undefined,
    winRate:
      realizedByJournal.length > 0
        ? roundToTwo((wins.length / realizedByJournal.length) * 100)
        : undefined,
    winningJournalCount: wins.length,
    completedJournalCount: realizedByJournal.length,
    profitLossRatio:
      wins.length > 0 && losses.length > 0
        ? roundToTwo(
            average(wins.map(({ realizedPnl }) => realizedPnl)) /
              Math.abs(average(losses.map(({ realizedPnl }) => realizedPnl))),
          )
        : undefined,
    averageWin: wins.length > 0 ? roundToTwo(average(wins.map(({ realizedPnl }) => realizedPnl))) : undefined,
    averageLoss:
      losses.length > 0
        ? roundToTwo(Math.abs(average(losses.map(({ realizedPnl }) => realizedPnl))))
        : undefined,
  };
}

function isSameDirection(quantity: number, currentNetPosition: number): boolean {
  return !isZero(quantity) && !isZero(currentNetPosition)
    ? Math.sign(quantity) === Math.sign(currentNetPosition)
    : false;
}

function realizedEventsForJournal(journal: TradeJournal): Array<{ timestamp: string; pnl: number }> {
  const orderedEvents = [...journal.events].sort(
    (left, right) => new Date(left.timestamp).getTime() - new Date(right.timestamp).getTime(),
  );
  const realizedEvents: Array<{ timestamp: string; pnl: number }> = [];
  let positionQuantity = 0;
  let averageCost = 0;

  for (const event of orderedEvents) {
    if (event.type === "log" || event.price === undefined || event.quantity === undefined) {
      continue;
    }

    if (positionQuantity === 0 || isSameDirection(event.quantity, positionQuantity)) {
      const nextQuantity = positionQuantity + event.quantity;
      averageCost =
        nextQuantity === 0
          ? 0
          : ((positionQuantity * averageCost) + (event.quantity * event.price)) / nextQuantity;
      positionQuantity = nextQuantity;
      continue;
    }

    const closingQuantity = Math.min(Math.abs(event.quantity), Math.abs(positionQuantity));
    const direction = Math.sign(positionQuantity);
    const pnl = direction * (event.price - averageCost) * closingQuantity;
    realizedEvents.push({ timestamp: event.timestamp, pnl: roundToTwo(pnl) });

    positionQuantity += event.quantity;
    if (positionQuantity === 0) {
      averageCost = 0;
    }
  }

  return realizedEvents;
}

function buildHeatmap(
  dailyMap: Map<string, number>,
  heatmapEndDayKey?: string,
  dayCount = HEATMAP_DAY_COUNT,
): DailyRealizedPnlHeatmapCell[] {
  const end = parseUtcDayKey(heatmapEndDayKey ?? toUtcDayKey(new Date().toISOString()));
  const start = addUtcDays(end, -(dayCount - 1));
  const visibleValues = Array.from(dailyMap.entries())
    .filter(([dayKey]) => dayKey >= toUtcDayKey(start.toISOString()) && dayKey <= toUtcDayKey(end.toISOString()))
    .map(([, value]) => Math.abs(value));
  const maxAbsValue = Math.max(0, ...visibleValues);

  return Array.from({ length: dayCount }, (_, index) => {
    const date = addUtcDays(start, index);
    const dayKey = toUtcDayKey(date.toISOString());
    const value = dailyMap.get(dayKey) ?? 0;

    return {
      dayKey,
      value,
      bucket: bucketForValue(value, maxAbsValue),
      direction: value > 0 ? "profit" : value < 0 ? "loss" : "flat",
    } satisfies DailyRealizedPnlHeatmapCell;
  });
}

function bucketForValue(value: number, maxAbsValue: number): number {
  if (isZero(value) || isZero(maxAbsValue)) {
    return 0;
  }

  return Math.max(1, Math.ceil((Math.abs(value) / maxAbsValue) * HEATMAP_BUCKETS));
}

function parseUtcDayKey(dayKey: string): Date {
  return new Date(`${dayKey}T00:00:00.000Z`);
}

function addUtcDays(date: Date, days: number): Date {
  return new Date(date.getTime() + days * 24 * 60 * 60 * 1000);
}

function average(values: number[]): number {
  return values.reduce((sum, value) => sum + value, 0) / values.length;
}
