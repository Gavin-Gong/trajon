import { isZero } from "../domain/shared";

export const REMINDER_INTERVALS = [
  { label: "1m", value: 60 },
  { label: "5m", value: 300 },
  { label: "15m", value: 900 },
  { label: "30m", value: 1800 },
  { label: "1h", value: 3600 },
];

export function formatClock(date: Date) {
  return date.toLocaleTimeString("en-GB", {
    hour12: false,
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

export function formatCalendarDate(date: Date) {
  const weekday = date.toLocaleDateString("zh-CN", { weekday: "short" });
  const monthDay = date.toLocaleDateString("en-CA", {
    month: "2-digit",
    day: "2-digit",
  });

  return `${weekday} ${monthDay}`;
}

export function formatReminderStatus(enabled: boolean, interval: number) {
  return enabled ? `开启 · ${formatReminderInterval(interval)}` : "关闭";
}

function formatReminderInterval(interval: number) {
  const match = REMINDER_INTERVALS.find((preset) => preset.value === interval);
  return match ? match.label : `${Math.round(interval / 60)}m`;
}

export function formatJournalDate(value: string) {
  return new Date(value).toLocaleString("zh-CN", {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

export function formatNumber(value: number) {
  return value.toLocaleString("en-US", {
    maximumFractionDigits: 2,
    minimumFractionDigits: isZero(value % 1) ? 0 : 0,
  });
}

export function formatSignedNumber(value: number) {
  const formatted = formatNumber(Math.abs(value));
  if (value > 0) {
    return `+${formatted}`;
  }

  if (value < 0) {
    return `-${formatted}`;
  }

  return formatted;
}

export function formatChartDateLabel(value: string) {
  return new Date(value).toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  });
}
