import "./setupDesktopTests";

import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { createJournal } from "../../domain/journal";
import * as PanelLifecycleModule from "../../platform/panelLifecycle";
import { parseAppView } from "../../platform/panelWindow";
import { Desktop } from "../Desktop";
import { buildState, expectTextPresent, resetSeedState, seedState } from "./desktopTestHelpers";

describe("TradingJournal panel", () => {
  beforeEach(() => {
    resetSeedState();
  });

  it("renders the panel surface with clock and Antd pin button", () => {
    render(<Desktop view={{ view: "panel" }} />);
    expect(screen.getByText("Quick Capture Panel")).toBeInTheDocument();
    expect(screen.getByTitle("当前保持置顶")).toBeInTheDocument();
    expect(document.querySelectorAll(".anticon-pushpin").length).toBeGreaterThanOrEqual(1);
  });

  it("shows the active journal summary in panel view", () => {
    render(<Desktop view={{ view: "panel" }} />);
    const panelMain = screen.getAllByRole("main")[0];
    expect(within(panelMain).getAllByText("NQ").length).toBeGreaterThanOrEqual(1);
    expect(within(panelMain).getAllByText("进行中").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByRole("button", { name: "结束 Journal" }).length).toBeGreaterThanOrEqual(1);
  });

  it("renders the requested journal when panel view receives journalId", () => {
    render(<Desktop view={{ view: "panel", journalId: "done-journal" }} />);
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

    render(<Desktop view={{ view: "panel", journalId: "done-journal", presetSymbol: "BTCUSDT" }} />);
    const panelMain = screen.getAllByRole("main")[0];
    expect(within(panelMain).getAllByText("ES").length).toBeGreaterThanOrEqual(1);
    expect(screen.queryByText("BTCUSDT")).not.toBeInTheDocument();
  });

  it("falls back to the current open journal when journalId is missing", () => {
    render(<Desktop view={{ view: "panel", journalId: "missing-journal" }} />);
    const panelMain = screen.getAllByRole("main")[0];
    expect(within(panelMain).getAllByText("NQ").length).toBeGreaterThanOrEqual(1);
    expect(screen.queryByText("纪律不错，但出场可以更果断。")).not.toBeInTheDocument();
  });

  it("renders the quick capture section with event type chips", () => {
    render(<Desktop view={{ view: "panel" }} />);
    expectTextPresent("快速记录");
    expect(screen.getAllByRole("radio", { name: "笔记" }).length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByRole("radio", { name: "加仓" }).length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByRole("radio", { name: "减仓" }).length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByRole("radio", { name: "平仓" }).length).toBeGreaterThanOrEqual(1);
  });

  it("uses segmented controls without card composition in panel view", () => {
    render(<Desktop view={{ view: "panel" }} />);
    expect(document.querySelectorAll(".ant-card")).toHaveLength(0);
    expect(document.querySelectorAll(".ant-segmented").length).toBeGreaterThanOrEqual(1);
    expect(document.querySelectorAll(".ant-empty").length).toBeGreaterThanOrEqual(0);
  });

  it("renders the panel without Ant Design cards", () => {
    render(<Desktop view={{ view: "panel" }} />);
    expect(document.querySelectorAll(".ant-card")).toHaveLength(0);
  });

  it("shows the timeline section with event count", () => {
    render(<Desktop view={{ view: "panel" }} />);
    expectTextPresent("最近记录");
  });

  it("shows reminder status in the panel header", () => {
    render(<Desktop view={{ view: "panel" }} />);
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

    render(<Desktop view={{ view: "panel" }} />);
    expect(screen.getByText("暂无总结")).toBeInTheDocument();
    expect(screen.queryByText("建议在收盘后补上执行结论与改进点。")).not.toBeInTheDocument();
    expect(screen.queryByText("把关键事件先记下来，复杂 dashboard 留到主界面复盘。")).not.toBeInTheDocument();
    expect(document.querySelectorAll(".ant-tooltip-open").length).toBe(0);
    expect(document.querySelectorAll(".anticon-info-circle").length).toBeGreaterThanOrEqual(2);
  });

  it("closes the panel after finishing the current journal", async () => {
    const closeSpy = vi.spyOn(PanelLifecycleModule, "closePanelWindow").mockResolvedValue(undefined);
    render(<Desktop view={{ view: "panel" }} />);
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

    render(<Desktop view={{ view: "panel" }} />);
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
    render(<Desktop view={{ view: "panel" }} />);
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
    render(<Desktop view={{ view: "panel" }} />);
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
    render(<Desktop view={view} />);
    expect(screen.getByText("ETHUSDT")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "开始记录今天的交易" })).not.toBeInTheDocument();
  });
});
