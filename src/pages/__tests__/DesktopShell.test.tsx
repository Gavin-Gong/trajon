import "./setupDesktopTests";

import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { openPanelWindow } from "../../platform/panelWindow";
import { Desktop } from "../Desktop";
import { Main } from "../Main";
import { TradingJournalPanel } from "../TradingJournalPanel";
import { buildState, expectTextPresent, resetSeedState, seedState } from "./desktopTestHelpers";

describe("Desktop shell", () => {
  beforeEach(() => {
    resetSeedState();
  });

  it("keeps the main and panel surfaces in dedicated component modules", () => {
    expect(Main).toBeTypeOf("function");
    expect(TradingJournalPanel).toBeTypeOf("function");
  });

  it("avoids handwritten padding and border radius styles in rendered surfaces", () => {
    seedState(buildState());
    const { rerender } = render(<Desktop view={{ view: "main" }} />);

    const mainRoot = screen.getAllByRole("main")[0];
    expect(mainRoot.getAttribute("style") ?? "").not.toContain("padding:");

    const sider = document.querySelector(".ant-layout-sider");
    expect(sider).not.toBeNull();
    expect(sider?.getAttribute("style") ?? "").not.toContain("padding:");

    const content = document.querySelector(".ant-layout-content");
    expect(content).not.toBeNull();
    expect(content?.getAttribute("style") ?? "").not.toContain("padding:");

    rerender(<Desktop view={{ view: "panel" }} />);

    const panelRoot = screen.getAllByRole("main")[0];
    expect(panelRoot.getAttribute("style") ?? "").not.toContain("padding:");
    expect(document.querySelectorAll(".ant-card")).toHaveLength(0);
  });

  it("avoids handwritten dark border styles in trading panel surfaces", () => {
    seedState(buildState());
    const { rerender } = render(<Desktop view={{ view: "main" }} />);

    const sider = document.querySelector(".ant-layout-sider");
    expect(sider).not.toBeNull();
    expect(sider?.getAttribute("style") ?? "").not.toContain("border-inline-end:");

    const divider = document.querySelector(".ant-divider");
    if (divider) {
      expect(divider.getAttribute("style") ?? "").not.toContain("border-color:");
    }

    rerender(<Desktop view={{ view: "panel" }} />);
    expect(document.querySelectorAll(".ant-card")).toHaveLength(0);
  });

  it("does not render removed app custom property styles in the desktop or panel views", () => {
    seedState(buildState());
    const { rerender, container } = render(<Desktop view={{ view: "main" }} />);

    for (const element of Array.from(container.querySelectorAll("[style]"))) {
      expect(element.getAttribute("style") ?? "").not.toContain("var(--");
    }

    rerender(<Desktop view={{ view: "panel" }} />);

    for (const element of Array.from(container.querySelectorAll("[style]"))) {
      expect(element.getAttribute("style") ?? "").not.toContain("var(--");
    }
  });

  it("renders the fixed shell header, branding, and navigation menu", () => {
    render(<Desktop view={{ view: "main" }} />);

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
    render(<Desktop view={{ view: "main" }} />);

    expect(document.querySelectorAll(".ant-layout").length).toBeGreaterThanOrEqual(1);
    expect(document.querySelectorAll(".ant-layout-sider").length).toBeGreaterThanOrEqual(1);
    expect(document.querySelectorAll(".ant-layout-content").length).toBeGreaterThanOrEqual(1);
    expect(document.querySelectorAll(".ant-card").length).toBeGreaterThanOrEqual(1);
  });

  it("avoids custom card radius and shadow overrides in the desktop shell", () => {
    render(<Desktop view={{ view: "main" }} />);

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
    render(<Desktop view={{ view: "main" }} />);
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

    render(<Desktop view={{ view: "main" }} />);
    fireEvent.click(screen.getByRole("button", { name: "更多交易对" }));
    fireEvent.click(await screen.findByRole("menuitem", { name: "开始交易 XAUUSD" }));

    expect(openPanelWindow).toHaveBeenCalledWith({ presetSymbol: "XAUUSD" });
  });

  it("hides the brand mark when the sider is collapsed", () => {
    render(<Desktop view={{ view: "main" }} />);
    expect(screen.getByTestId("trading-journal-brand-mark")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "收起导航" }));
    expect(screen.queryByTestId("trading-journal-brand-mark")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "展开导航" })).toBeInTheDocument();
  });
});
