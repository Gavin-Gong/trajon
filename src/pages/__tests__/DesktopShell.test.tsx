import "./setupDesktopTests";

import { fireEvent, render, screen, waitFor } from "@testing-library/react";
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
    expect(screen.getByText("查看累计已实现盈亏与每日盈亏分布。")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "开始记录" })).toBeInTheDocument();
    expect(screen.getByLabelText("进行中的 Journal")).toBeInTheDocument();
    const menus = screen.getAllByRole("menu");
    expect(menus.length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByRole("menuitem").length).toBeGreaterThanOrEqual(3);
    expectTextPresent("面板");
    expectTextPresent("日历");
    expectTextPresent("设置");
    expect(screen.queryByText("Dashboard")).not.toBeInTheDocument();
    expect(screen.queryByText("分析")).not.toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: /Chart/ })).not.toBeInTheDocument();
  });

  it("places the in-progress badge trigger to the right of the start button", () => {
    render(<Desktop view={{ view: "main" }} />);

    const startButton = screen.getByRole("button", { name: "开始记录" });
    const journalTrigger = screen.getByLabelText("进行中的 Journal");
    const headerActions = startButton.parentElement;

    expect(headerActions).not.toBeNull();
    expect(headerActions?.children[0]).toBe(startButton);
    expect(headerActions?.children[1]).toBe(journalTrigger.closest(".ant-badge"));
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
    fireEvent.click(screen.getByRole("button", { name: "开始记录" }));
    expect(openPanelWindow).toHaveBeenCalledWith();
  });

  it("opens a drawer with in-progress journals from the header badge action", () => {
    render(<Desktop view={{ view: "main" }} />);

    fireEvent.click(screen.getByLabelText("进行中的 Journal"));

    expect(screen.getByText("进行中的 Journal")).toBeInTheDocument();
    expect(screen.getByText("NQ")).toBeInTheDocument();
    expect(screen.queryByText("ES")).not.toBeInTheDocument();
  });

  it("opens the panel window for a selected in-progress journal", () => {
    render(<Desktop view={{ view: "main" }} />);

    fireEvent.click(screen.getByLabelText("进行中的 Journal"));
    fireEvent.click(screen.getByRole("button", { name: "继续记录-journal-open-journal" }));

    expect(openPanelWindow).toHaveBeenCalledWith({ journalId: "open-journal" });
  });

  it("updates the main badge when a panel finishes the open journal", async () => {
    render(
      <>
        <Desktop view={{ view: "main" }} />
        <Desktop view={{ view: "panel", journalId: "open-journal" }} />
      </>,
    );

    fireEvent.click(screen.getAllByLabelText("进行中的 Journal")[0]!);
    expect(screen.getByRole("button", { name: "继续记录-journal-open-journal" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("radio", { name: "复盘总结" }));
    fireEvent.change(await screen.findByRole("textbox", { name: "复盘" }), {
      target: { value: "按计划结束" },
    });
    fireEvent.click(screen.getByRole("button", { name: "完成复盘" }));

    await waitFor(() => {
      expect(screen.queryByRole("button", { name: "继续记录-journal-open-journal" })).not.toBeInTheDocument();
    });
  });

  it("hides the brand mark when the sider is collapsed", () => {
    render(<Desktop view={{ view: "main" }} />);
    expect(screen.getByTestId("trading-journal-brand-mark")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "收起导航" }));
    expect(screen.queryByTestId("trading-journal-brand-mark")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "展开导航" })).toBeInTheDocument();
  });
});
