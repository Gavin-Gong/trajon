import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MainSurface } from "./components/tradingPanel/TradingPanelMain";
import { GLOBAL_CONFIG_FALLBACK_STORAGE_KEY } from "./globalConfigStorage";
import { TRADING_PANEL_STORAGE_KEY, type TradingPanelState } from "./model/storage";
import { PanelSurface } from "./components/tradingPanel/TradingPanelPanel";
import * as PanelLifecycleModule from "./panelLifecycle";
import { composeTradeEvent, createJournal } from "./tradingPanelDomain";
import { TradingPanelDesktop } from "./tradingPanelApp";
import { parseAppView } from "./panelWindow";
import { openPanelWindow } from "./panelWindow";

vi.mock("./panelWindow", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./panelWindow")>();
  return {
    ...actual,
    openPanelWindow: vi.fn(() => Promise.resolve()),
  };
});

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
    pairPresets: [],
    defaultPresetSymbol: undefined,
    chart: {
      symbol: "BINANCE:BTCUSDT",
      interval: "60",
    },
    panelPinned: true,
  };
}

function seedState(state: TradingPanelState) {
  window.localStorage.setItem(
    TRADING_PANEL_STORAGE_KEY,
    JSON.stringify({
      journals: state.journals,
      chart: state.chart,
      panelPinned: state.panelPinned,
    }),
  );
  window.localStorage.setItem(
    GLOBAL_CONFIG_FALLBACK_STORAGE_KEY,
    JSON.stringify({
      config: state.config,
      pairPresets: state.pairPresets,
      defaultPresetSymbol: state.defaultPresetSymbol,
    }),
  );
}

function expectTextPresent(text: string | RegExp) {
  const matches = screen.getAllByText(text);
  expect(matches.length).toBeGreaterThanOrEqual(1);
}

function openAntdSelect(testId: string) {
  const root = screen.getByTestId(testId);
  const trigger = root.querySelector('[role="combobox"]');
  expect(trigger).not.toBeNull();
  fireEvent.mouseDown(trigger as Element);
}

function chooseAntdOption(name: string) {
  fireEvent.click(screen.getByRole("option", { name }));
}

function typeIntoAntdAutoComplete(testId: string, value: string) {
  const root = screen.getByTestId(testId);
  const input = root.matches("input") ? root : root.querySelector("input");
  expect(input).not.toBeNull();
  fireEvent.change(input as Element, { target: { value } });
}

describe("TradingPanelDesktop", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("keeps the main and panel surfaces in dedicated component modules", () => {
    expect(MainSurface).toBeTypeOf("function");
    expect(PanelSurface).toBeTypeOf("function");
  });

  it("avoids handwritten padding and border radius styles in rendered surfaces", () => {
    seedState(buildState());
    const { rerender } = render(<TradingPanelDesktop view={{ view: "main" }} />);

    const mainRoot = screen.getAllByRole("main")[0];
    expect(mainRoot.getAttribute("style") ?? "").not.toContain("padding:");

    const sider = document.querySelector(".ant-layout-sider");
    expect(sider).not.toBeNull();
    expect(sider?.getAttribute("style") ?? "").not.toContain("padding:");

    const content = document.querySelector(".ant-layout-content");
    expect(content).not.toBeNull();
    expect(content?.getAttribute("style") ?? "").not.toContain("padding:");

    rerender(<TradingPanelDesktop view={{ view: "panel" }} />);

    const panelRoot = screen.getAllByRole("main")[0];
    expect(panelRoot.getAttribute("style") ?? "").not.toContain("padding:");

    const handwrittenStyledCards = Array.from(document.querySelectorAll(".ant-card[style]"));
    expect(handwrittenStyledCards.length).toBeGreaterThan(0);
    for (const card of handwrittenStyledCards) {
      const style = card.getAttribute("style") ?? "";
      expect(style).not.toContain("border-radius:");
    }
  });

  it("avoids handwritten dark border styles in trading panel surfaces", () => {
    seedState(buildState());
    const { rerender } = render(<TradingPanelDesktop view={{ view: "main" }} />);

    const sider = document.querySelector(".ant-layout-sider");
    expect(sider).not.toBeNull();
    expect(sider?.getAttribute("style") ?? "").not.toContain("border-inline-end:");

    const divider = document.querySelector(".ant-divider");
    if (divider) {
      expect(divider.getAttribute("style") ?? "").not.toContain("border-color:");
    }

    rerender(<TradingPanelDesktop view={{ view: "panel" }} />);

    const styledCards = Array.from(document.querySelectorAll(".ant-card[style]"));
    expect(styledCards.length).toBeGreaterThan(0);
    for (const card of styledCards) {
      const style = card.getAttribute("style") ?? "";
      expect(style).not.toContain("var(--app-border)");
      expect(style).not.toContain("var(--app-border-strong)");
      expect(style).not.toContain("border-style: dashed");
    }
  });

  describe("main view – desktop shell structure", () => {
    it("renders an empty analysis placeholder while keeping the desktop shell", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view={{ view: "main" }} />);

      expectTextPresent("分析页面暂时留空");
      expectTextPresent("后续会在这里逐步补回分析模块。");
    });

    it("renders the sidebar branding and navigation menu", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view={{ view: "main" }} />);

      expect(screen.getAllByText("Trading Flow").length).toBeGreaterThanOrEqual(1);

      const menus = screen.getAllByRole("menu");
      expect(menus.length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByRole("menuitem").length).toBeGreaterThanOrEqual(4);
      expectTextPresent("分析");
      expectTextPresent("Chart");
      expectTextPresent("日历");
      expectTextPresent("设置");
    });

    it("switches to the chart section when the chart menu item is clicked", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view={{ view: "main" }} />);

      fireEvent.click(screen.getByRole("menuitem", { name: /Chart/ }));

      expect(screen.getByText("图表工作台")).toBeInTheDocument();
      expect(screen.queryByText("分析页面暂时留空")).not.toBeInTheDocument();
    });

    it("renders symbol and interval controls in the chart section", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view={{ view: "main" }} />);

      fireEvent.click(screen.getByRole("menuitem", { name: /Chart/ }));

      expect(screen.getByText("选择标的")).toBeInTheDocument();
      expect(screen.getByText("时间周期")).toBeInTheDocument();
      expect(screen.getByText("BINANCE:BTCUSDT")).toBeInTheDocument();
      expect(screen.getAllByText("60").length).toBeGreaterThanOrEqual(1);
    });

    it("includes AAPL and Kweichow Moutai in chart symbol options", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view={{ view: "main" }} />);

      fireEvent.click(screen.getByRole("menuitem", { name: /Chart/ }));

      openAntdSelect("chart-symbol-select");

      expect(screen.getByRole("option", { name: "NASDAQ:AAPL" })).toBeInTheDocument();
      expect(screen.getByRole("option", { name: "SSE:600519" })).toBeInTheDocument();
    });

    it("renders a stable TradingView widget container", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view={{ view: "main" }} />);

      fireEvent.click(screen.getByRole("menuitem", { name: /Chart/ }));

      expect(screen.getByTestId("tradingview-widget-shell")).toBeInTheDocument();
      expect(screen.getByTestId("tradingview-widget-container")).toBeInTheDocument();
    });

    it("gives the TradingView autosize container an explicit height", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view={{ view: "main" }} />);

      fireEvent.click(screen.getByRole("menuitem", { name: /Chart/ }));

      expect(screen.getByTestId("tradingview-widget-container")).toHaveStyle({
        height: "520px",
        overflow: "hidden",
      });
    });

    it("updates the chart widget configuration when symbol changes", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view={{ view: "main" }} />);

      fireEvent.click(screen.getByRole("menuitem", { name: /Chart/ }));
      openAntdSelect("chart-symbol-select");
      chooseAntdOption("BINANCE:ETHUSDT");

      expect(within(screen.getByTestId("chart-symbol-select")).getByText("BINANCE:ETHUSDT")).toBeInTheDocument();
      expect(screen.getByTestId("tradingview-widget-status")).toHaveTextContent(
        "当前图表: BINANCE:ETHUSDT / 60",
      );
    });

    it("injects the TradingView advanced chart embed script", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view={{ view: "main" }} />);

      fireEvent.click(screen.getByRole("menuitem", { name: /Chart/ }));

      const widgetScript = screen
        .getByTestId("tradingview-widget-container")
        .querySelector('script[src="https://s3.tradingview.com/external-embedding/embed-widget-advanced-chart.js"]');

      expect(widgetScript).not.toBeNull();
    });

    it("restores persisted chart symbol and interval when opening the chart page", () => {
      seedState({
        ...buildState(),
        chart: {
          symbol: "BINANCE:ETHUSDT",
          interval: "240",
        },
      });

      render(<TradingPanelDesktop view={{ view: "main" }} />);
      fireEvent.click(screen.getByRole("menuitem", { name: /Chart/ }));

      expect(screen.getByText("BINANCE:ETHUSDT")).toBeInTheDocument();
      expect(screen.getByText("240")).toBeInTheDocument();
      expect(screen.getByTestId("tradingview-widget-status")).toHaveTextContent(
        "当前图表: BINANCE:ETHUSDT / 240",
      );
    });

    it("restores newly added stock symbols when reopening the chart page", () => {
      seedState({
        ...buildState(),
        chart: {
          symbol: "NASDAQ:AAPL",
          interval: "1D",
        },
      });

      render(<TradingPanelDesktop view={{ view: "main" }} />);
      fireEvent.click(screen.getByRole("menuitem", { name: /Chart/ }));

      expect(screen.getByText("NASDAQ:AAPL")).toBeInTheDocument();
      expect(screen.getByText("1D")).toBeInTheDocument();
      expect(screen.getByTestId("tradingview-widget-status")).toHaveTextContent(
        "当前图表: NASDAQ:AAPL / 1D",
      );
    });

    it("uses Ant Design layout and card composition for the desktop workspace", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view={{ view: "main" }} />);

      expect(document.querySelectorAll(".ant-layout").length).toBeGreaterThanOrEqual(1);
      expect(document.querySelectorAll(".ant-layout-sider").length).toBeGreaterThanOrEqual(1);
      expect(document.querySelectorAll(".ant-layout-content").length).toBeGreaterThanOrEqual(1);
      expect(document.querySelectorAll(".ant-card").length).toBeGreaterThanOrEqual(1);
    });

    it("avoids custom card radius and shadow overrides in the desktop shell", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view={{ view: "main" }} />);

      const cards = Array.from(document.querySelectorAll(".ant-card"));
      expect(cards.length).toBeGreaterThanOrEqual(1);
      for (const card of cards) {
        const style = card.getAttribute("style") ?? "";
        expect(style).not.toContain("border-radius: 28px");
        expect(style).not.toContain("box-shadow: var(--app-shadow-soft)");
        expect(style).not.toContain("background: var(--app-surface-strong)");
      }
    });

    it("renders the float button for opening the panel window", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view={{ view: "main" }} />);

      // FloatButton tooltip text is lazily rendered in jsdom, so we assert on the button element itself
      const floatButtons = document.querySelectorAll(".ant-float-btn");
      expect(floatButtons.length).toBeGreaterThanOrEqual(1);
    });
  });

  describe("main view – analysis page", () => {
    it("shows the same empty placeholder regardless of journal state", () => {
      seedState({
        journals: [],
        config: { timerEnabled: false, timerInterval: 300 },
        pairPresets: [],
        defaultPresetSymbol: undefined,
        chart: {
          symbol: "BINANCE:BTCUSDT",
          interval: "60",
        },
        panelPinned: false,
      });
      render(<TradingPanelDesktop view={{ view: "main" }} />);

      expectTextPresent("分析页面暂时留空");
      expectTextPresent("后续会在这里逐步补回分析模块。");
      expect(document.querySelectorAll(".ant-empty").length).toBeGreaterThanOrEqual(1);
      expect(screen.queryByText("盘中复盘工作台")).not.toBeInTheDocument();
    });

    it("lets the user choose a preset in analytics before opening panel", () => {
      seedState({
        journals: [],
        config: { timerEnabled: false, timerInterval: 300 },
        pairPresets: [
          { symbol: "BTCUSDT", timeframe: "5m" },
          { symbol: "ETHUSDT", timeframe: "15m" },
        ],
        defaultPresetSymbol: "BTCUSDT",
        chart: {
          symbol: "BINANCE:BTCUSDT",
          interval: "60",
        },
        panelPinned: false,
      });

      render(<TradingPanelDesktop view={{ view: "main" }} />);

      openAntdSelect("analytics-panel-preset-select");
      chooseAntdOption("ETHUSDT · 15m");
      fireEvent.click(screen.getByRole("button", { name: "使用所选预设打开 Panel" }));

      expect(openPanelWindow).toHaveBeenCalledWith({
        presetSymbol: "ETHUSDT",
      });
    });

    it("does not render the temporary fs demo card on the analysis page", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view={{ view: "main" }} />);

      expect(screen.queryByText("Tauri FS Demo")).not.toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "写入示例" })).not.toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "读取示例" })).not.toBeInTheDocument();
    });
  });

  describe("panel view – entry point", () => {
    it("renders the panel surface with clock and Antd pin button", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view={{ view: "panel" }} />);

      expect(screen.getByText("Quick Capture Panel")).toBeInTheDocument();
      expect(screen.getByTitle("当前保持置顶")).toBeInTheDocument();
      expect(document.querySelectorAll(".anticon-pushpin").length).toBeGreaterThanOrEqual(1);
    });

    it("shows the active journal summary in panel view", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view={{ view: "panel" }} />);

      const mains = screen.getAllByRole("main");
      expect(mains.length).toBeGreaterThanOrEqual(1);
      const panelMain = mains[0];
      expect(within(panelMain).getAllByText("NQ").length).toBeGreaterThanOrEqual(1);
      expect(within(panelMain).getAllByText("进行中").length).toBeGreaterThanOrEqual(1);

      expect(screen.getAllByRole("button", { name: "结束 Journal" }).length).toBeGreaterThanOrEqual(1);
    });

    it("renders the quick capture section with event type chips", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view={{ view: "panel" }} />);

      expectTextPresent("快速记录");
      expect(screen.getAllByRole("radio", { name: "笔记" }).length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByRole("radio", { name: "加仓" }).length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByRole("radio", { name: "减仓" }).length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByRole("radio", { name: "平仓" }).length).toBeGreaterThanOrEqual(1);
    });

    it("uses Ant Design cards, segmented controls, and list composition in panel view", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view={{ view: "panel" }} />);

      expect(document.querySelectorAll(".ant-card").length).toBeGreaterThanOrEqual(3);
      expect(document.querySelectorAll(".ant-segmented").length).toBeGreaterThanOrEqual(1);
      expect(document.querySelectorAll(".ant-empty").length).toBeGreaterThanOrEqual(0);
    });

    it("avoids custom card radius overrides in panel view", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view={{ view: "panel" }} />);

      const cards = Array.from(document.querySelectorAll(".ant-card"));
      expect(cards.length).toBeGreaterThanOrEqual(1);
      for (const card of cards) {
        const style = card.getAttribute("style") ?? "";
        expect(style).not.toContain("border-radius: 24px");
        expect(style).not.toContain("border-radius: 28px");
      }
    });

    it("shows the timeline section with event count", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view={{ view: "panel" }} />);

      expectTextPresent("最近记录");
    });

    it("shows reminder status in the panel header", () => {
      seedState(buildState());
      render(<TradingPanelDesktop view={{ view: "panel" }} />);

      expectTextPresent(/开启 · 5m/);
    });

    it("closes the panel after finishing the current journal", async () => {
      seedState(buildState());
      const closeSpy = vi
        .spyOn(PanelLifecycleModule, "closePanelWindow")
        .mockResolvedValue(undefined);

      render(<TradingPanelDesktop view={{ view: "panel" }} />);
      fireEvent.click(screen.getByRole("button", { name: "结束 Journal" }));
      fireEvent.change(screen.getByPlaceholderText("可选：这笔交易哪里做得好，哪里需要改进？"), {
        target: { value: "Finished and closed." },
      });
      fireEvent.click(screen.getByRole("button", { name: "确认结束" }));

      await waitFor(() => {
        expect(closeSpy).toHaveBeenCalledTimes(1);
      });
      closeSpy.mockRestore();
    });
  });

  describe("panel view – auto-created journal", () => {
    it("creates a new journal immediately when the latest journal is already done", () => {
      seedState({
        journals: [
          createJournal({
            id: "done-only",
            symbol: "BTCUSDT",
            market: "Binance",
            createdAt: "2026-04-10T09:30:00.000Z",
            status: "done",
            summary: "Closed the session.",
          }),
        ],
        config: { timerEnabled: false, timerInterval: 300 },
        pairPresets: [{ symbol: "BTCUSDT", timeframe: "5m" }],
        defaultPresetSymbol: "BTCUSDT",
        chart: {
          symbol: "BINANCE:BTCUSDT",
          interval: "60",
        },
        panelPinned: false,
      });

      render(<TradingPanelDesktop view={{ view: "panel" }} />);

      expect(screen.getByText("BTCUSDT")).toBeInTheDocument();
      expect(screen.queryByText("Closed the session.")).not.toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "开始记录今天的交易" })).not.toBeInTheDocument();
      expect(screen.getByText(/未标记市场/)).toBeInTheDocument();
    });

    it("creates a new journal immediately when no journals exist", () => {
      seedState({
        journals: [],
        config: { timerEnabled: false, timerInterval: 300 },
        pairPresets: [{ symbol: "BTCUSDT", timeframe: "5m" }],
        defaultPresetSymbol: "BTCUSDT",
        chart: {
          symbol: "BINANCE:BTCUSDT",
          interval: "60",
        },
        panelPinned: false,
      });
      render(<TradingPanelDesktop view={{ view: "panel" }} />);

      expect(screen.getByText("BTCUSDT")).toBeInTheDocument();
      expect(screen.queryByText("没有活跃 Journal")).not.toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "开始记录今天的交易" })).not.toBeInTheDocument();
    });

    it("unlocks quick capture immediately when no journals exist", () => {
      seedState({
        journals: [],
        config: { timerEnabled: false, timerInterval: 300 },
        pairPresets: [{ symbol: "BTCUSDT", timeframe: "5m" }],
        defaultPresetSymbol: "BTCUSDT",
        chart: {
          symbol: "BINANCE:BTCUSDT",
          interval: "60",
        },
        panelPinned: false,
      });
      render(<TradingPanelDesktop view={{ view: "panel" }} />);

      expect(screen.queryByText(/先创建今天的 journal，快速记录区才会解锁/)).not.toBeInTheDocument();
      expect(screen.getByRole("button", { name: "记录事件" })).toBeInTheDocument();
    });

    it("prefers presetSymbol from panel URL when auto-creating a journal", () => {
      seedState({
        journals: [],
        config: { timerEnabled: false, timerInterval: 300 },
        pairPresets: [
          { symbol: "BTCUSDT", timeframe: "5m" },
          { symbol: "ETHUSDT", timeframe: "15m" },
        ],
        defaultPresetSymbol: "BTCUSDT",
        chart: {
          symbol: "BINANCE:BTCUSDT",
          interval: "60",
        },
        panelPinned: false,
      });

      const view = parseAppView("?view=panel&presetSymbol=ETHUSDT");
      render(<TradingPanelDesktop view={view} />);

      expect(screen.getByText("ETHUSDT")).toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "开始记录今天的交易" })).not.toBeInTheDocument();
    });
  });

  describe("main view – settings presets", () => {
    it("renders trading pair preset controls in settings", () => {
      seedState({
        journals: [],
        config: { timerEnabled: false, timerInterval: 300 },
        pairPresets: [{ symbol: "BTCUSDT", timeframe: "5m" }],
        defaultPresetSymbol: "BTCUSDT",
        chart: {
          symbol: "BINANCE:BTCUSDT",
          interval: "60",
        },
        panelPinned: true,
      });

      render(<TradingPanelDesktop view={{ view: "main" }} />);
      fireEvent.click(screen.getByRole("menuitem", { name: /设置/ }));

      expect(screen.getByText("交易对预设")).toBeInTheDocument();
      expect(screen.getByText("BTCUSDT")).toBeInTheDocument();
      expect(screen.getAllByText("5m").length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByText(/默认预设/).length).toBeGreaterThanOrEqual(1);
    });

    it("saves a typed trading pair preset with timeframe from settings", () => {
      seedState({
        journals: [],
        config: { timerEnabled: false, timerInterval: 300 },
        pairPresets: [],
        defaultPresetSymbol: undefined,
        chart: {
          symbol: "BINANCE:BTCUSDT",
          interval: "60",
        },
        panelPinned: true,
      });

      render(<TradingPanelDesktop view={{ view: "main" }} />);
      fireEvent.click(screen.getByRole("menuitem", { name: /设置/ }));

      typeIntoAntdAutoComplete("pair-preset-symbol-input", "ethusdt");
      openAntdSelect("pair-preset-timeframe-select");
      chooseAntdOption("15m");
      fireEvent.click(screen.getByRole("button", { name: "保存为预设" }));

      const presetList = screen.getByTestId("pair-preset-list");
      expect(within(presetList).getByText("ETHUSDT")).toBeInTheDocument();
      expect(within(presetList).getAllByText("15m").length).toBeGreaterThanOrEqual(1);
    });

    it("updates an existing trading pair preset timeframe from settings", () => {
      seedState({
        journals: [],
        config: { timerEnabled: false, timerInterval: 300 },
        pairPresets: [{ symbol: "BTCUSDT", timeframe: "5m" }],
        defaultPresetSymbol: undefined,
        chart: {
          symbol: "BINANCE:BTCUSDT",
          interval: "60",
        },
        panelPinned: true,
      });

      render(<TradingPanelDesktop view={{ view: "main" }} />);
      fireEvent.click(screen.getByRole("menuitem", { name: /设置/ }));

      typeIntoAntdAutoComplete("pair-preset-symbol-input", "btcusdt");
      openAntdSelect("pair-preset-timeframe-select");
      chooseAntdOption("15m");
      fireEvent.click(screen.getByRole("button", { name: "保存为预设" }));

      const presetList = screen.getByTestId("pair-preset-list");
      expect(within(presetList).getAllByText("BTCUSDT")).toHaveLength(1);
      expect(within(presetList).getAllByText("15m").length).toBeGreaterThanOrEqual(1);
    });

    it("prefills the default trading pair preset when opening panel without an active journal", () => {
      seedState({
        journals: [],
        config: { timerEnabled: false, timerInterval: 300 },
        pairPresets: [{ symbol: "BTCUSDT", timeframe: "5m" }],
        defaultPresetSymbol: "BTCUSDT",
        chart: {
          symbol: "BINANCE:BTCUSDT",
          interval: "60",
        },
        panelPinned: false,
      });

      render(<TradingPanelDesktop view={{ view: "panel" }} />);

      expect(screen.getByText("BTCUSDT")).toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "开始记录今天的交易" })).not.toBeInTheDocument();
    });

    it("lets the user set a saved preset as default from settings", async () => {
      seedState({
        journals: [],
        config: { timerEnabled: false, timerInterval: 300 },
        pairPresets: [
          { symbol: "BTCUSDT", timeframe: "5m" },
          { symbol: "ETHUSDT", timeframe: "15m" },
        ],
        defaultPresetSymbol: "BTCUSDT",
        chart: {
          symbol: "BINANCE:BTCUSDT",
          interval: "60",
        },
        panelPinned: false,
      });

      const { unmount } = render(<TradingPanelDesktop view={{ view: "main" }} />);
      fireEvent.click(screen.getByRole("menuitem", { name: /设置/ }));
      fireEvent.click(screen.getByRole("button", { name: "设为默认-ETHUSDT" }));

      expect(within(screen.getByTestId("pair-preset-list")).getAllByText(/默认预设/).length).toBeGreaterThanOrEqual(1);
      await waitFor(() => {
        expect(window.localStorage.getItem(GLOBAL_CONFIG_FALLBACK_STORAGE_KEY)).toContain("ETHUSDT");
      });

      unmount();
      render(<TradingPanelDesktop view={{ view: "panel" }} />);

      await expect(screen.findByText("ETHUSDT")).resolves.toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "开始记录今天的交易" })).not.toBeInTheDocument();
    });
  });
});
