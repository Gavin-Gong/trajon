import { render, screen, within } from "@testing-library/react";
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

function seedState(state: TradingPanelState) {
  window.localStorage.setItem(TRADING_PANEL_STORAGE_KEY, JSON.stringify(state));
}

function expectTextPresent(text: string | RegExp) {
  const matches = screen.getAllByText(text);
  expect(matches.length).toBeGreaterThanOrEqual(1);
}

describe("TradingPanelDesktop", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  describe("main view – desktop shell structure", () => {
    it("renders an empty analysis placeholder while keeping the desktop shell", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view="main" />);

      expectTextPresent("分析页面暂时留空");
      expectTextPresent("后续会在这里逐步补回分析模块。");
    });

    it("renders the sidebar branding and navigation menu", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view="main" />);

      expect(screen.getAllByText("Trading Flow").length).toBeGreaterThanOrEqual(1);

      const menus = screen.getAllByRole("menu");
      expect(menus.length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByRole("menuitem").length).toBeGreaterThanOrEqual(3);
      expectTextPresent("分析");
      expectTextPresent("日历");
      expectTextPresent("设置");
    });

    it("uses Ant Design layout and card composition for the desktop workspace", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view="main" />);

      expect(document.querySelectorAll(".ant-layout").length).toBeGreaterThanOrEqual(1);
      expect(document.querySelectorAll(".ant-layout-sider").length).toBeGreaterThanOrEqual(1);
      expect(document.querySelectorAll(".ant-layout-content").length).toBeGreaterThanOrEqual(1);
      expect(document.querySelectorAll(".ant-card").length).toBeGreaterThanOrEqual(1);
    });

    it("renders the float button for opening the panel window", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view="main" />);

      // FloatButton tooltip text is lazily rendered in jsdom, so we assert on the button element itself
      const floatButtons = document.querySelectorAll(".ant-float-btn");
      expect(floatButtons.length).toBeGreaterThanOrEqual(1);
    });
  });

  describe("main view – analysis placeholder", () => {
    it("shows the same empty placeholder regardless of journal state", () => {
      seedState({
        journals: [],
        config: { timerEnabled: false, timerInterval: 300 },
        panelPinned: false,
      });
      render(<TradingPanelDesktop view="main" />);

      expectTextPresent("分析页面暂时留空");
      expectTextPresent("后续会在这里逐步补回分析模块。");
      expect(document.querySelectorAll(".ant-empty").length).toBeGreaterThanOrEqual(1);
      expect(screen.queryByText("盘中复盘工作台")).not.toBeInTheDocument();
    });
  });

  describe("panel view – entry point", () => {
    it("renders the panel surface with clock and Antd pin button", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view="panel" />);

      expect(screen.getByText("Quick Capture Panel")).toBeInTheDocument();
      expect(screen.getByTitle("当前保持置顶")).toBeInTheDocument();
      expect(document.querySelectorAll(".anticon-pushpin").length).toBeGreaterThanOrEqual(1);
    });

    it("shows the active journal summary in panel view", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view="panel" />);

      const mains = screen.getAllByRole("main");
      expect(mains.length).toBeGreaterThanOrEqual(1);
      const panelMain = mains[0];
      expect(within(panelMain).getAllByText("NQ").length).toBeGreaterThanOrEqual(1);
      expect(within(panelMain).getAllByText("进行中").length).toBeGreaterThanOrEqual(1);

      expect(screen.getAllByRole("button", { name: "结束 Journal" }).length).toBeGreaterThanOrEqual(1);
    });

    it("renders the quick capture section with event type chips", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view="panel" />);

      expectTextPresent("快速记录");
      expect(screen.getAllByRole("radio", { name: "笔记" }).length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByRole("radio", { name: "加仓" }).length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByRole("radio", { name: "减仓" }).length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByRole("radio", { name: "平仓" }).length).toBeGreaterThanOrEqual(1);
    });

    it("uses Ant Design cards, segmented controls, and list composition in panel view", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view="panel" />);

      expect(document.querySelectorAll(".ant-card").length).toBeGreaterThanOrEqual(3);
      expect(document.querySelectorAll(".ant-segmented").length).toBeGreaterThanOrEqual(1);
      expect(document.querySelectorAll(".ant-empty").length).toBeGreaterThanOrEqual(0);
    });

    it("shows the timeline section with event count", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view="panel" />);

      expectTextPresent("最近记录");
    });

    it("shows reminder status in the panel header", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view="panel" />);

      expectTextPresent(/开启 · 5m/);
    });
  });

  describe("panel view – empty state (no journals)", () => {
    it("shows the create journal prompt when no journals exist", () => {
      seedState({
        journals: [],
        config: { timerEnabled: false, timerInterval: 300 },
        panelPinned: false,
      });
      render(<TradingPanelDesktop view="panel" />);

      expect(screen.getByText("没有活跃 Journal")).toBeInTheDocument();
      expect(
        screen.getByText(/先创建今天的交易记录/),
      ).toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: "开始记录今天的交易" }),
      ).toBeInTheDocument();
    });

    it("shows locked quick capture area when no journals exist", () => {
      seedState({
        journals: [],
        config: { timerEnabled: false, timerInterval: 300 },
        panelPinned: false,
      });
      render(<TradingPanelDesktop view="panel" />);

      expectTextPresent(/先创建今天的 journal，快速记录区才会解锁/);
      expect(screen.getAllByRole("button", { name: "开始 Journal" }).length).toBeGreaterThanOrEqual(1);
    });
  });
});
