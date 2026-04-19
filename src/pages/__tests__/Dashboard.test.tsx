import "./setupDesktopTests";

import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { openPanelWindow } from "../../platform/panelWindow";
import { Desktop } from "../Desktop";
import { buildState, resetSeedState, seedState } from "./desktopTestHelpers";

describe("Dashboard", () => {
  beforeEach(() => {
    resetSeedState();
  });

  it("renders an empty dashboard placeholder while keeping the desktop shell", () => {
    render(<Desktop view={{ view: "main" }} />);
  });

  it("shows the same empty placeholder regardless of journal state", () => {
    seedState({
      journals: [],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      panelPinned: false,
    });
    render(<Desktop view={{ view: "main" }} />);
    expect(screen.queryByText("盘中复盘工作台")).not.toBeInTheDocument();
  });

  it("uses a narrow right rail for recent journal cards", () => {
    render(<Desktop view={{ view: "main" }} />);
    expect(screen.getByTestId("dashboard-panel-layout").getAttribute("style") ?? "").toContain(
      "grid-template-columns: minmax(0, 1fr) 280px",
    );
  });

  it("renders two placeholder cards on the left side of dashboard", () => {
    render(<Desktop view={{ view: "main" }} />);
    expect(screen.getByTestId("dashboard-placeholder-card-1")).toBeInTheDocument();
    expect(screen.getByTestId("dashboard-placeholder-card-2")).toBeInTheDocument();
  });

  it("keeps the recent journals card compact without description copy", () => {
    render(<Desktop view={{ view: "main" }} />);
    expect(screen.queryByText("继续补记观察，或直接回到上一条交易会话。")).not.toBeInTheDocument();
  });

  it("uses an icon-only edit button for recent journals", () => {
    render(<Desktop view={{ view: "main" }} />);
    const editButton = screen.getByRole("button", { name: "编辑-journal-done-journal" });
    expect(editButton).toBeInTheDocument();
    expect(editButton).not.toHaveTextContent("继续编辑");
  });

  it("renders the journal list inside a single right-side card", () => {
    render(<Desktop view={{ view: "main" }} />);
    expect(screen.getByTestId("dashboard-journal-list-card")).toBeInTheDocument();
    expect(screen.queryByTestId("dashboard-journal-card-open-journal")).not.toBeInTheDocument();
    expect(screen.queryByTestId("dashboard-journal-card-done-journal")).not.toBeInTheDocument();
  });

  it("shows updated time instead of market and net position in the journal list", () => {
    render(<Desktop view={{ view: "main" }} />);
    expect(screen.getAllByText(/更新于/).length).toBeGreaterThanOrEqual(1);
    expect(screen.queryByText("CME")).not.toBeInTheDocument();
    expect(screen.queryByText(/净仓/)).not.toBeInTheDocument();
  });

  it("does not render the temporary fs demo card on the dashboard page", () => {
    render(<Desktop view={{ view: "main" }} />);
    expect(screen.queryByText("Tauri FS Demo")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "写入示例" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "读取示例" })).not.toBeInTheDocument();
  });

  it("opens the panel with journalId from dashboard continue action", () => {
    seedState(buildState());
    render(<Desktop view={{ view: "main" }} />);
    fireEvent.click(screen.getByRole("button", { name: "编辑-journal-done-journal" }));
    expect(openPanelWindow).toHaveBeenCalledWith({ journalId: "done-journal" });
  });
});
