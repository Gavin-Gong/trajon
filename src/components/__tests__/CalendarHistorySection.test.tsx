import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { createJournal } from "../../domain/journal";
import { CalendarHistorySection } from "../CalendarHistorySection";

vi.mock("../../platform/journalImages", () => ({
  isJournalImageManagementAvailable: () => false,
  pickJournalImageFiles: vi.fn(),
  resolveJournalImageSrc: vi.fn(async (relativePath: string) => `asset://${relativePath}`),
}));

describe("CalendarHistorySection", () => {
  it("opens the shared journal drawer when clicking a calendar journal", async () => {
    const journal = createJournal({
      id: "calendar-journal",
      symbol: "BTCUSDT",
      createdAt: "2026-04-10T09:30:00.000Z",
      updatedAt: "2026-04-10T09:30:00.000Z",
      status: "open",
      summary: "Breakout setup.",
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
        finishJournal: vi.fn().mockResolvedValue(undefined),
        recordEvent: vi.fn(),
        addJournalImages: vi.fn(),
        removeJournalImage: vi.fn(),
        deleteJournal: vi.fn(),
        updateConfig: vi.fn(),
        upsertPairPreset: vi.fn(),
        removePairPreset: vi.fn(),
        setDefaultPresetSymbol: vi.fn(),
        setPanelPinned: vi.fn(),
        refreshState: vi.fn(),
        updateJournalSymbol: vi.fn().mockResolvedValue(undefined),
      },
    };

    render(<CalendarHistorySection store={store as never} />);

    fireEvent.click(screen.getByRole("button", { name: "BTCUSDT" }));

    await waitFor(() => {
      expect(screen.getByText("快速记录")).toBeInTheDocument();
    });
    expect(screen.getByText("最近记录")).toBeInTheDocument();
  });
});
