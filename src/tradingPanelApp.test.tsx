import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { composeTradeEvent, createJournal } from "./tradingPanelDomain";
import { TradingPanelDesktop } from "./tradingPanelApp";
import { TRADING_PANEL_STORAGE_KEY, type TradingPanelState } from "./tradingPanelStore";

function buildState(): TradingPanelState {
  const openJournal = createJournal({
    id: "open-journal",
    symbol: "NQ",
    market: "CME",
    createdAt: "2026-04-04T09:30:00.000Z",
    status: "open",
    summary: "等待二次确认后继续执行。",
  });
  const doneJournal = createJournal({
    id: "done-journal",
    symbol: "ES",
    market: "CME",
    createdAt: "2026-04-02T09:30:00.000Z",
    status: "done",
    summary: "纪律不错，但出场可以更果断。",
  });

  const entry = composeTradeEvent(
    {
      type: "entry",
      content: "Opening drive long",
      quantityInput: "2",
      priceInput: "18250",
    },
    0,
    "open",
  );
  const log = composeTradeEvent(
    {
      type: "log",
      content: "保持耐心，等待回踩确认。",
      quantityInput: "",
      priceInput: "",
    },
    2,
    "open",
  );

  expect(entry.ok).toBe(true);
  expect(log.ok).toBe(true);

  if (entry.ok && log.ok) {
    openJournal.events = [entry.event, log.event];
  }

  return {
    journals: [doneJournal, openJournal],
    config: {
      timerEnabled: true,
      timerInterval: 300,
    },
    panelPinned: true,
  };
}

describe("TradingPanelDesktop", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("renders the upgraded light analytics workspace for intraday review", () => {
    window.localStorage.setItem(TRADING_PANEL_STORAGE_KEY, JSON.stringify(buildState()));

    render(<TradingPanelDesktop view="main" />);

    expect(screen.getByText("盘中复盘工作台")).toBeInTheDocument();
    expect(screen.getByText("今日执行概览")).toBeInTheDocument();
    expect(screen.getByText("本月日志沉淀")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "打开记录面板" })).toBeInTheDocument();
    expect(screen.getByText("NQ")).toBeInTheDocument();
  });
});
