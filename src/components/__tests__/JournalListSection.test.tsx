import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { createJournal } from "../../domain/journal";
import { JournalListSection } from "../JournalListSection";

vi.mock("../../platform/journalImages", () => ({
  isJournalImageManagementAvailable: () => false,
  pickJournalImageFiles: vi.fn(),
  resolveJournalImageSrc: vi.fn(async (relativePath: string) => `asset://${relativePath}`),
}));

describe("JournalListSection", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("filters journals by any selected tag", async () => {
    const breakoutJournal = createJournal({
      id: "journal-breakout",
      symbol: "BTCUSDT",
      status: "done",
      summary: "Breakout trade.",
      tags: ["breakout", "trend"],
    });
    const fomoJournal = createJournal({
      id: "journal-fomo",
      symbol: "ETHUSDT",
      status: "done",
      summary: "Fomo chase.",
      tags: ["FOMO"],
    });
    const calmJournal = createJournal({
      id: "journal-calm",
      symbol: "SOLUSDT",
      status: "done",
      summary: "No tags here.",
      tags: [],
    });

    const store = {
      state: {
        journals: [breakoutJournal, fomoJournal, calmJournal],
        config: { timerEnabled: false, timerInterval: 300 },
        pairPresets: [],
        defaultPresetSymbol: undefined,
        panelPinned: true,
      },
      actions: {
        createJournal: vi.fn(),
        finishJournal: vi.fn(),
        recordEvent: vi.fn(),
        addJournalImages: vi.fn(),
        removeJournalImage: vi.fn(),
        deleteJournal: vi.fn(),
        updateJournalSymbol: vi.fn(),
        updateConfig: vi.fn(),
        upsertPairPreset: vi.fn(),
        removePairPreset: vi.fn(),
        setDefaultPresetSymbol: vi.fn(),
        setPanelPinned: vi.fn(),
        refreshState: vi.fn(),
      },
    };

    render(<JournalListSection store={store} />);

    expect(screen.getByText("共 3 条")).toBeInTheDocument();

    fireEvent.mouseDown(screen.getByLabelText("按标签筛选"));
    fireEvent.click(screen.getByTitle("breakout"));
    fireEvent.mouseDown(screen.getByLabelText("按标签筛选"));
    fireEvent.click(screen.getByTitle("FOMO"));

    await waitFor(() => {
      expect(screen.getByText("共 2 条")).toBeInTheDocument();
    });

    expect(screen.getByText("BTCUSDT")).toBeInTheDocument();
    expect(screen.getByText("ETHUSDT")).toBeInTheDocument();
    expect(screen.queryByText("SOLUSDT")).not.toBeInTheDocument();
  });

  it("opens shared drawer content from edit action", async () => {
    const openJournal = createJournal({
      id: "open-journal",
      symbol: "BTCUSDT",
      status: "open",
      summary: "Breakout setup.",
    });
    const store = {
      state: {
        journals: [openJournal],
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
        deleteJournal: vi.fn().mockResolvedValue(undefined),
        updateConfig: vi.fn(),
        upsertPairPreset: vi.fn(),
        removePairPreset: vi.fn(),
        setDefaultPresetSymbol: vi.fn(),
        setPanelPinned: vi.fn(),
        refreshState: vi.fn(),
        updateJournalSymbol: vi.fn().mockResolvedValue(undefined),
      },
    };

    render(<JournalListSection store={store as never} />);

    fireEvent.click(screen.getByRole("button", { name: /编辑/ }));

    await waitFor(() => {
      expect(screen.getByText("快速记录")).toBeInTheDocument();
    });
  });
});
