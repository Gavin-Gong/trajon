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
    const orderedEvents = [...journal.events].sort(
      (left, right) => new Date(left.timestamp).getTime() - new Date(right.timestamp).getTime(),
    );

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

function isSameDirection(quantity: number, currentNetPosition: number): boolean {
  return !isZero(quantity) && !isZero(currentNetPosition)
    ? Math.sign(quantity) === Math.sign(currentNetPosition)
    : false;
}

function buildHeatmap(
  dailyMap: Map<string, number>,
  heatmapEndDayKey?: string,
): DailyRealizedPnlHeatmapCell[] {
  const end = parseUtcDayKey(heatmapEndDayKey ?? toUtcDayKey(new Date().toISOString()));
  const start = addUtcDays(end, -(HEATMAP_DAY_COUNT - 1));
  const visibleValues = Array.from(dailyMap.entries())
    .filter(([dayKey]) => dayKey >= toUtcDayKey(start.toISOString()) && dayKey <= toUtcDayKey(end.toISOString()))
    .map(([, value]) => Math.abs(value));
  const maxAbsValue = Math.max(0, ...visibleValues);

  return Array.from({ length: HEATMAP_DAY_COUNT }, (_, index) => {
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
