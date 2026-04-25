import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { createJournal } from "../../domain/journal";
import { JournalDrawer } from "../JournalDrawer";

vi.mock("../../platform/journalImages", () => ({
  isJournalImageManagementAvailable: () => false,
  pickJournalImageFiles: vi.fn(),
  resolveJournalImageSrc: vi.fn(async (relativePath: string) => `asset://${relativePath}`),
}));

describe("JournalDrawer", () => {
  it("closes the drawer after finishing review", async () => {
    const openJournal = createJournal({
      id: "open-journal",
      symbol: "NQ",
      market: "CME",
      status: "open",
      events: [],
      images: [],
    });
    const onClose = vi.fn();
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
        recordEvent: vi.fn().mockResolvedValue({ ok: true, event: { id: "event-1" } }),
        addJournalImages: vi.fn().mockResolvedValue(undefined),
        removeJournalImage: vi.fn().mockResolvedValue(undefined),
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

    render(<JournalDrawer open journalId="open-journal" store={store as never} onClose={onClose} />);

    expect(screen.getByText("快速记录")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "复 盘" }));
    fireEvent.change(screen.getByRole("textbox", { name: "复盘" }), {
      target: { value: "纪律执行到位" },
    });
    fireEvent.click(screen.getByRole("button", { name: /完\s*成/ }));

    await waitFor(() => {
      expect(store.actions.finishJournal).toHaveBeenCalledWith("open-journal", "纪律执行到位", []);
    });
    await waitFor(() => {
      expect(onClose).toHaveBeenCalledTimes(1);
    });
  });
});
