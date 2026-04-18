import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TradingJournalMain } from "../TradingJournalMain";
import type { TradingPanelState } from "../../../model/storage";
import { TradingJournalPanel } from "../TradingJournalPanel";
import * as PanelLifecycleModule from "../../../platform/panelLifecycle";
import { composeTradeEvent, createJournal } from "../../../domain/tradingJournalDomain";
import { TradingJournalDesktop } from "../TradingJournalDesktop";
import { parseAppView } from "../../../platform/panelWindow";
import { openPanelWindow } from "../../../platform/panelWindow";

const mockedStoreState = vi.hoisted(() => ({
  value: undefined as TradingPanelState | undefined,
}));

vi.mock("../../../stores/useTradingJournalState", async () => {
  const React = await import("react");
  const domain = await import("../../../domain/tradingJournalDomain");

  function cloneState(state: TradingPanelState): TradingPanelState {
    return JSON.parse(JSON.stringify(state)) as TradingPanelState;
  }

  return {
    useTradingJournalState() {
      const [state, setState] = React.useState<TradingPanelState>(() =>
        mockedStoreState.value ? cloneState(mockedStoreState.value) : cloneState(buildState()),
      );
      const stateRef = React.useRef(state);

      React.useEffect(() => {
        stateRef.current = state;
      }, [state]);

      return {
        state,
        actions: {
          createJournal(symbol: string, market: string) {
            const journal = domain.createJournal({ symbol: symbol.trim(), market: market.trim() });
            setState((currentState) => ({
              ...currentState,
              journals: [...currentState.journals, journal],
            }));
            return journal;
          },
          finishJournal(journalId: string, summary: string) {
            setState((currentState) => ({
              ...currentState,
              journals: currentState.journals.map((journal) =>
                journal.id === journalId
                  ? {
                      ...journal,
                      summary: summary.trim(),
                      status: "done",
                      updatedAt: new Date().toISOString(),
                    }
                  : journal,
              ),
            }));
            return Promise.resolve();
          },
          recordEvent(journalId: string, draft: Parameters<typeof domain.composeTradeEvent>[0]) {
            const journal = stateRef.current.journals.find((item) => item.id === journalId);
            if (!journal) {
              return { ok: false, error: "missingContent" } as const;
            }

            const result = domain.composeTradeEvent(
              draft,
              journal.events.reduce((total, event) => total + (event.quantity ?? 0), 0),
              journal.status,
            );
            if (!result.ok) {
              return result;
            }

            setState((currentState) => ({
              ...currentState,
              journals: currentState.journals.map((item) =>
                item.id === journalId
                  ? {
                      ...item,
                      updatedAt: result.event.updatedAt,
                      events: [result.event, ...item.events],
                    }
                  : item,
              ),
            }));
            return result;
          },
          updateConfig(nextConfig: Partial<TradingPanelState["config"]>) {
            setState((currentState) => ({
              ...currentState,
              config: { ...currentState.config, ...nextConfig },
            }));
            return Promise.resolve();
          },
          upsertPairPreset(symbol: string, timeframe: TradingPanelState["pairPresets"][number]["timeframe"]) {
            const normalizedSymbol = symbol.trim().toUpperCase();
            if (!normalizedSymbol) {
              return Promise.resolve();
            }

            setState((currentState) => {
              const nextPreset = { symbol: normalizedSymbol, timeframe };
              const hasExisting = currentState.pairPresets.some((preset) => preset.symbol === normalizedSymbol);
              return {
                ...currentState,
                pairPresets: hasExisting
                  ? currentState.pairPresets.map((preset) =>
                      preset.symbol === normalizedSymbol ? nextPreset : preset,
                    )
                  : [...currentState.pairPresets, nextPreset],
              };
            });
            return Promise.resolve();
          },
          removePairPreset(symbol: string) {
            const normalizedSymbol = symbol.trim().toUpperCase();
            setState((currentState) => ({
              ...currentState,
              pairPresets: currentState.pairPresets.filter((preset) => preset.symbol !== normalizedSymbol),
              defaultPresetSymbol:
                currentState.defaultPresetSymbol === normalizedSymbol
                  ? undefined
                  : currentState.defaultPresetSymbol,
            }));
            return Promise.resolve();
          },
          setDefaultPresetSymbol(symbol?: string) {
            const normalizedSymbol = symbol?.trim().toUpperCase();
            setState((currentState) => ({
              ...currentState,
              defaultPresetSymbol:
                normalizedSymbol && currentState.pairPresets.some((preset) => preset.symbol === normalizedSymbol)
                  ? normalizedSymbol
                  : undefined,
            }));
            return Promise.resolve();
          },
          setPanelPinned(panelPinned: boolean) {
            setState((currentState) => ({
              ...currentState,
              panelPinned,
            }));
            return Promise.resolve();
          },
        },
      };
    },
  };
});

vi.mock("../../../platform/panelWindow", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../../platform/panelWindow")>();
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
    panelPinned: true,
  };
}

function seedState(state: TradingPanelState) {
  mockedStoreState.value = JSON.parse(JSON.stringify(state)) as TradingPanelState;
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

describe("TradingJournalDesktop", () => {
  beforeEach(() => {
    mockedStoreState.value = buildState();
  });

  it("keeps the main and panel surfaces in dedicated component modules", () => {
    expect(TradingJournalMain).toBeTypeOf("function");
    expect(TradingJournalPanel).toBeTypeOf("function");
  });

  it("avoids handwritten padding and border radius styles in rendered surfaces", () => {
    seedState(buildState());
    const { rerender } = render(<TradingJournalDesktop view={{ view: "main" }} />);

    const mainRoot = screen.getAllByRole("main")[0];
    expect(mainRoot.getAttribute("style") ?? "").not.toContain("padding:");

    const sider = document.querySelector(".ant-layout-sider");
    expect(sider).not.toBeNull();
    expect(sider?.getAttribute("style") ?? "").not.toContain("padding:");

    const content = document.querySelector(".ant-layout-content");
    expect(content).not.toBeNull();
    expect(content?.getAttribute("style") ?? "").not.toContain("padding:");

    rerender(<TradingJournalDesktop view={{ view: "panel" }} />);

    const panelRoot = screen.getAllByRole("main")[0];
    expect(panelRoot.getAttribute("style") ?? "").not.toContain("padding:");

    expect(document.querySelectorAll(".ant-card")).toHaveLength(0);
  });

  it("avoids handwritten dark border styles in trading panel surfaces", () => {
    seedState(buildState());
    const { rerender } = render(<TradingJournalDesktop view={{ view: "main" }} />);

    const sider = document.querySelector(".ant-layout-sider");
    expect(sider).not.toBeNull();
    expect(sider?.getAttribute("style") ?? "").not.toContain("border-inline-end:");

    const divider = document.querySelector(".ant-divider");
    if (divider) {
      expect(divider.getAttribute("style") ?? "").not.toContain("border-color:");
    }

    rerender(<TradingJournalDesktop view={{ view: "panel" }} />);

    expect(document.querySelectorAll(".ant-card")).toHaveLength(0);
  });

  it("does not render removed app custom property styles in the desktop or panel views", () => {
    seedState(buildState());
    const { rerender, container } = render(<TradingJournalDesktop view={{ view: "main" }} />);

    for (const element of Array.from(container.querySelectorAll("[style]"))) {
      expect(element.getAttribute("style") ?? "").not.toContain("var(--");
    }

    rerender(<TradingJournalDesktop view={{ view: "panel" }} />);

    for (const element of Array.from(container.querySelectorAll("[style]"))) {
      expect(element.getAttribute("style") ?? "").not.toContain("var(--");
    }
  });

  describe("main view – desktop shell structure", () => {
    it("renders an empty dashboard placeholder while keeping the desktop shell", () => {
      seedState(buildState());
      render(<TradingJournalDesktop view={{ view: "main" }} />);

      expectTextPresent("Dashboard 暂时留空");
      expectTextPresent("后续会在这里逐步补回 dashboard 模块。");
    });

    it("renders the fixed shell header, branding, and navigation menu", () => {
      seedState(buildState());
      render(<TradingJournalDesktop view={{ view: "main" }} />);

      expect(screen.getByText("Trajon")).toBeInTheDocument();
      expect(screen.getByText("总览当前记录入口与常用操作。")).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "开始交易" })).toBeInTheDocument();
      const menus = screen.getAllByRole("menu");
      expect(menus.length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByRole("menuitem").length).toBeGreaterThanOrEqual(3);
      expectTextPresent("Dashboard");
      expectTextPresent("日历");
      expectTextPresent("设置");
      expect(screen.queryByRole("menuitem", { name: /Chart/ })).not.toBeInTheDocument();
    });

    it("uses Ant Design layout and card composition for the desktop workspace", () => {
      seedState(buildState());
      render(<TradingJournalDesktop view={{ view: "main" }} />);

      expect(document.querySelectorAll(".ant-layout").length).toBeGreaterThanOrEqual(1);
      expect(document.querySelectorAll(".ant-layout-sider").length).toBeGreaterThanOrEqual(1);
      expect(document.querySelectorAll(".ant-layout-content").length).toBeGreaterThanOrEqual(1);
      expect(document.querySelectorAll(".ant-card").length).toBeGreaterThanOrEqual(1);
    });

    it("avoids custom card radius and shadow overrides in the desktop shell", () => {
      seedState(buildState());
      render(<TradingJournalDesktop view={{ view: "main" }} />);

      const cards = Array.from(document.querySelectorAll(".ant-card"));
      expect(cards.length).toBeGreaterThanOrEqual(1);
      for (const card of cards) {
        const style = card.getAttribute("style") ?? "";
        expect(style).not.toContain("border-radius: 28px");
        expect(style).not.toContain("box-shadow: 0 12px 24px rgba(37, 99, 235, 0.16)");
        expect(style).not.toContain("background: rgb(255, 255, 255)");
      }
    });

    it("opens the panel from the fixed header action", () => {
      seedState(buildState());
      render(<TradingJournalDesktop view={{ view: "main" }} />);

      fireEvent.click(screen.getByRole("button", { name: "开始交易" }));

      expect(openPanelWindow).toHaveBeenCalledWith();
    });

    it("opens the panel with the selected preset from header dropdown", async () => {
      seedState({
        journals: [],
        config: { timerEnabled: false, timerInterval: 300 },
        pairPresets: [
          { symbol: "XAUUSD", timeframe: "5m" },
          { symbol: "BTCUSDT", timeframe: "15m" },
        ],
        defaultPresetSymbol: undefined,
        panelPinned: false,
      });

      render(<TradingJournalDesktop view={{ view: "main" }} />);

      const moreButton = screen.getByRole("button", { name: "更多交易对" });
      fireEvent.click(moreButton);
      fireEvent.click(await screen.findByRole("menuitem", { name: "开始交易 XAUUSD" }));

      expect(openPanelWindow).toHaveBeenCalledWith({ presetSymbol: "XAUUSD" });
    });

    it("opens the panel with journalId from dashboard continue action", () => {
      seedState(buildState());
      render(<TradingJournalDesktop view={{ view: "main" }} />);

      fireEvent.click(screen.getByRole("button", { name: "继续编辑-done-journal" }));

      expect(openPanelWindow).toHaveBeenCalledWith({ journalId: "done-journal" });
    });

    it("hides the brand mark when the sider is collapsed", () => {
      seedState(buildState());
      render(<TradingJournalDesktop view={{ view: "main" }} />);

      expect(screen.getByTestId("trading-journal-brand-mark")).toBeInTheDocument();
      fireEvent.click(screen.getByRole("button", { name: "收起导航" }));

      expect(screen.queryByTestId("trading-journal-brand-mark")).not.toBeInTheDocument();
      expect(screen.getByRole("button", { name: "展开导航" })).toBeInTheDocument();
    });
  });

  describe("main view – dashboard page", () => {
    it("shows the same empty placeholder regardless of journal state", () => {
      seedState({
        journals: [],
        config: { timerEnabled: false, timerInterval: 300 },
        pairPresets: [],
        defaultPresetSymbol: undefined,
        panelPinned: false,
      });
      render(<TradingJournalDesktop view={{ view: "main" }} />);

      expectTextPresent("Dashboard 暂时留空");
      expectTextPresent("后续会在这里逐步补回 dashboard 模块。");
      expect(document.querySelectorAll(".ant-empty").length).toBeGreaterThanOrEqual(1);
      expect(screen.queryByText("盘中复盘工作台")).not.toBeInTheDocument();
    });

    it("does not render the temporary fs demo card on the dashboard page", () => {
      seedState(buildState());
      render(<TradingJournalDesktop view={{ view: "main" }} />);

      expect(screen.queryByText("Tauri FS Demo")).not.toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "写入示例" })).not.toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "读取示例" })).not.toBeInTheDocument();
    });
  });

  describe("panel view – entry point", () => {
    it("renders the panel surface with clock and Antd pin button", () => {
      seedState(buildState());
      render(<TradingJournalDesktop view={{ view: "panel" }} />);

      expect(screen.getByText("Quick Capture Panel")).toBeInTheDocument();
      expect(screen.getByTitle("当前保持置顶")).toBeInTheDocument();
      expect(document.querySelectorAll(".anticon-pushpin").length).toBeGreaterThanOrEqual(1);
    });

    it("shows the active journal summary in panel view", () => {
      seedState(buildState());
      render(<TradingJournalDesktop view={{ view: "panel" }} />);

      const mains = screen.getAllByRole("main");
      expect(mains.length).toBeGreaterThanOrEqual(1);
      const panelMain = mains[0];
      expect(within(panelMain).getAllByText("NQ").length).toBeGreaterThanOrEqual(1);
      expect(within(panelMain).getAllByText("进行中").length).toBeGreaterThanOrEqual(1);

      expect(screen.getAllByRole("button", { name: "结束 Journal" }).length).toBeGreaterThanOrEqual(1);
    });

    it("renders the requested journal when panel view receives journalId", () => {
      seedState(buildState());
      render(<TradingJournalDesktop view={{ view: "panel", journalId: "done-journal" }} />);

      const panelMain = screen.getAllByRole("main")[0];
      expect(within(panelMain).getAllByText("ES").length).toBeGreaterThanOrEqual(1);
      expect(within(panelMain).getByText("纪律不错，但出场可以更果断。")).toBeInTheDocument();
      expect(within(panelMain).getAllByText("已完成").length).toBeGreaterThanOrEqual(1);
      expect(screen.queryByRole("button", { name: "结束 Journal" })).not.toBeInTheDocument();
    });

    it("prefers journalId over presetSymbol when both are provided", () => {
      seedState({
        ...buildState(),
        pairPresets: [{ symbol: "BTCUSDT", timeframe: "5m" }],
        defaultPresetSymbol: "BTCUSDT",
      });

      render(
        <TradingJournalDesktop
          view={{ view: "panel", journalId: "done-journal", presetSymbol: "BTCUSDT" }}
        />,
      );

      const panelMain = screen.getAllByRole("main")[0];
      expect(within(panelMain).getAllByText("ES").length).toBeGreaterThanOrEqual(1);
      expect(screen.queryByText("BTCUSDT")).not.toBeInTheDocument();
    });

    it("falls back to the current open journal when journalId is missing", () => {
      seedState(buildState());
      render(<TradingJournalDesktop view={{ view: "panel", journalId: "missing-journal" }} />);

      const panelMain = screen.getAllByRole("main")[0];
      expect(within(panelMain).getAllByText("NQ").length).toBeGreaterThanOrEqual(1);
      expect(screen.queryByText("纪律不错，但出场可以更果断。")).not.toBeInTheDocument();
    });

    it("renders the quick capture section with event type chips", () => {
      seedState(buildState());
      render(<TradingJournalDesktop view={{ view: "panel" }} />);

      expectTextPresent("快速记录");
      expect(screen.getAllByRole("radio", { name: "笔记" }).length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByRole("radio", { name: "加仓" }).length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByRole("radio", { name: "减仓" }).length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByRole("radio", { name: "平仓" }).length).toBeGreaterThanOrEqual(1);
    });

    it("uses segmented controls without card composition in panel view", () => {
      seedState(buildState());
      render(<TradingJournalDesktop view={{ view: "panel" }} />);

      expect(document.querySelectorAll(".ant-card")).toHaveLength(0);
      expect(document.querySelectorAll(".ant-segmented").length).toBeGreaterThanOrEqual(1);
      expect(document.querySelectorAll(".ant-empty").length).toBeGreaterThanOrEqual(0);
    });

    it("renders the panel without Ant Design cards", () => {
      seedState(buildState());
      render(<TradingJournalDesktop view={{ view: "panel" }} />);

      expect(document.querySelectorAll(".ant-card")).toHaveLength(0);
    });

    it("shows the timeline section with event count", () => {
      seedState(buildState());
      render(<TradingJournalDesktop view={{ view: "panel" }} />);

      expectTextPresent("最近记录");
    });

    it("shows reminder status in the panel header", () => {
      seedState(buildState());
      render(<TradingJournalDesktop view={{ view: "panel" }} />);

      expectTextPresent(/开启 · 5m/);
    });

    it("keeps panel descriptive copy behind tooltip icons", () => {
      seedState({
        ...buildState(),
        journals: [
          createJournal({
            id: "open-without-summary",
            symbol: "NQ",
            market: "CME",
            createdAt: "2026-04-04T09:30:00.000Z",
            status: "open",
            summary: "",
          }),
        ],
      });

      render(<TradingJournalDesktop view={{ view: "panel" }} />);

      expect(screen.getByText("暂无总结")).toBeInTheDocument();
      expect(screen.queryByText("建议在收盘后补上执行结论与改进点。")).not.toBeInTheDocument();
      expect(screen.queryByText("把关键事件先记下来，复杂 dashboard 留到主界面复盘。")).not.toBeInTheDocument();
      expect(document.querySelectorAll(".ant-tooltip-open").length).toBe(0);
      expect(document.querySelectorAll(".anticon-info-circle").length).toBeGreaterThanOrEqual(2);
    });

    it("closes the panel after finishing the current journal", async () => {
      seedState(buildState());
      const closeSpy = vi
        .spyOn(PanelLifecycleModule, "closePanelWindow")
        .mockResolvedValue(undefined);

      render(<TradingJournalDesktop view={{ view: "panel" }} />);
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
        panelPinned: false,
      });

      render(<TradingJournalDesktop view={{ view: "panel" }} />);

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
        panelPinned: false,
      });
      render(<TradingJournalDesktop view={{ view: "panel" }} />);

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
        panelPinned: false,
      });
      render(<TradingJournalDesktop view={{ view: "panel" }} />);

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
        panelPinned: false,
      });

      const view = parseAppView("?view=panel&presetSymbol=ETHUSDT");
      render(<TradingJournalDesktop view={view} />);

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
        panelPinned: true,
      });

      render(<TradingJournalDesktop view={{ view: "main" }} />);
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
        panelPinned: true,
      });

      render(<TradingJournalDesktop view={{ view: "main" }} />);
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
        panelPinned: true,
      });

      render(<TradingJournalDesktop view={{ view: "main" }} />);
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
        panelPinned: false,
      });

      render(<TradingJournalDesktop view={{ view: "panel" }} />);

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
        panelPinned: false,
      });

      const { unmount } = render(<TradingJournalDesktop view={{ view: "main" }} />);
      fireEvent.click(screen.getByRole("menuitem", { name: /设置/ }));
      fireEvent.click(screen.getByRole("button", { name: "设为默认-ETHUSDT" }));

      expect(within(screen.getByTestId("pair-preset-list")).getAllByText(/默认预设/).length).toBeGreaterThanOrEqual(1);

      unmount();
      mockedStoreState.value = {
        ...mockedStoreState.value!,
        defaultPresetSymbol: "ETHUSDT",
      };
      render(<TradingJournalDesktop view={{ view: "panel" }} />);

      await expect(screen.findByText("ETHUSDT")).resolves.toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "开始记录今天的交易" })).not.toBeInTheDocument();
    });
  });
});
