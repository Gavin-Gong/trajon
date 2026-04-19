export function isZero(value: number, tolerance = 0.000001): boolean {
  return Math.abs(value) < tolerance;
}

export function toUtcDayKey(value: string): string {
  const date = new Date(value);
  const year = date.getUTCFullYear();
  const month = `${date.getUTCMonth() + 1}`.padStart(2, "0");
  const day = `${date.getUTCDate()}`.padStart(2, "0");
  return `${year}-${month}-${day}`;
}

export function roundToTwo(value: number): number {
  return Number(value.toFixed(2));
}
