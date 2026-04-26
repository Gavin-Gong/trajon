import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { createJournal } from "../../domain/journal";
import { AnalysisSection } from "../AnalysisSection";

vi.mock("@ant-design/plots", () => ({
  Line: () => <div data-testid="line-chart" />,
  Column: () => <div data-testid="column-chart" />,
}));

describe("AnalysisSection", () => {
  it("shows a tooltip with day and pnl when hovering a heatmap cell", async () => {
    const journal = createJournal({
      id: "journal-analysis",
      symbol: "BTCUSDT",
      status: "done",
      events: [
        {
          id: "entry-1",
          createdAt: "2026-04-18T09:00:00.000Z",
          updatedAt: "2026-04-18T09:00:00.000Z",
          timestamp: "2026-04-18T09:00:00.000Z",
          type: "entry",
          content: "Open",
          price: 100,
          quantity: 1,
        },
        {
          id: "exit-1",
          createdAt: "2026-04-18T10:00:00.000Z",
          updatedAt: "2026-04-18T10:00:00.000Z",
          timestamp: "2026-04-18T10:00:00.000Z",
          type: "exit",
          content: "Close",
          price: 115,
          quantity: -1,
        },
      ],
    });

    const store = {
      state: {
        journals: [journal],
        config: { timerEnabled: false, timerInterval: 300 },
        pairPresets: [],
        defaultPresetSymbol: undefined,
        panelPinned: true,
      },
      actions: {
        createJournal: vi.fn(),
        finishJournal: vi.fn(),
        addJournalImages: vi.fn(),
        removeJournalImage: vi.fn(),
        deleteJournal: vi.fn(),
        updateJournalSymbol: vi.fn(),
        recordEvent: vi.fn(),
        updateConfig: vi.fn(),
        upsertPairPreset: vi.fn(),
        removePairPreset: vi.fn(),
        setDefaultPresetSymbol: vi.fn(),
        setPanelPinned: vi.fn(),
        refreshState: vi.fn(),
      },
    };

    render(<AnalysisSection store={store} />);

    const heatmapCell = screen.getByRole("gridcell", { name: "2026-04-18 +15" });
    fireEvent.mouseEnter(heatmapCell);

    expect(await screen.findByText("2026-04-18")).toBeInTheDocument();
    expect(await screen.findByText("当日已实现盈亏 +15")).toBeInTheDocument();
  });
});
