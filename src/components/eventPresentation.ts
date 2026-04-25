import type { TradeEventType } from "../domain/journal";

import type { AppTagVariant } from "./AppTypes";

export const EVENT_LABELS: Record<TradeEventType, string> = {
  entry: "开仓",
  add: "加仓",
  reduce: "减仓",
  exit: "平仓",
  log: "笔记",
  plan: "计划",
};

export const EVENT_SYMBOLS: Record<TradeEventType, string> = {
  entry: "↗",
  add: "+",
  reduce: "−",
  exit: "×",
  log: "☰",
  plan: "◌",
};

export function tagColorForVariant(variant: AppTagVariant) {
  switch (variant) {
    case "accent":
      return "blue";
    case "success":
      return "green";
    case "warning":
      return "orange";
    case "danger":
      return "red";
    case "neutral":
      return undefined;
  }
}

export function eventVariant(type: TradeEventType): AppTagVariant {
  switch (type) {
    case "entry":
      return "success";
    case "add":
    case "reduce":
      return "warning";
    case "exit":
      return "danger";
    case "log":
      return "accent";
    case "plan":
      return "neutral";
  }

  return "neutral";
}
