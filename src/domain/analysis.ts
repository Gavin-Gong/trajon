import type { TradeJournal } from "./journal";
import { isZero, roundToTwo, toUtcDayKey } from "./shared";

export type RealizedPnlPoint = {
  timestamp: string;
  value: number;
};

export type DailyRealizedPnlPoint = {
  dayKey: string;
  value: number;
};

export type RealizedPnlAnalytics = {
  realizedTotal: number;
  equityCurve: RealizedPnlPoint[];
  dailyPnl: DailyRealizedPnlPoint[];
  profitDays: number;
  lossDays: number;
  journalCount: number;
};

export function realizedPnlAnalytics(
  journals: TradeJournal[],
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

  return {
    realizedTotal,
    equityCurve,
    dailyPnl,
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
