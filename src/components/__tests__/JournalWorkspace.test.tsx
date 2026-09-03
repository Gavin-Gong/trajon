import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { createJournal } from "../../domain/journal";
import type { TradeJournal } from "../../domain/journal";
import { JournalWorkspace } from "../JournalWorkspace";

vi.mock("../../platform/journalImages", () => ({
  isJournalImageManagementAvailable: () => false,
  pickJournalImageFiles: vi.fn(),
  resolveJournalImageSrc: vi.fn(async (relativePath: string) => `asset://${relativePath}`),
}));

describe("JournalWorkspace", () => {
  it("renders the capture step by default and keeps saved review content visible", () => {
    const journal = createJournal({
      id: "journal-1",
      symbol: "NQ",
      status: "open",
      summary: "已经做过首轮复盘",
      tags: ["纪律", "执行"],
      images: [],
      events: [],
    });

    render(<JournalWorkspace journal={journal} store={createStore(journal)} isPinned={false} onTogglePinned={vi.fn()} />);

    expect(screen.getByText("快速记录")).toBeInTheDocument();
    expect(screen.getByText("已经做过首轮复盘")).toBeInTheDocument();
    expect(screen.getByText("纪律")).toBeInTheDocument();
    expect(screen.getByText("执行")).toBeInTheDocument();
  });

  it("renders a pill segmented header", () => {
    const journal = createJournal({
      id: "journal-steps",
      symbol: "NQ",
      status: "open",
      events: [],
      images: [],
    });

    render(<JournalWorkspace journal={journal} store={createStore(journal)} isPinned={false} onTogglePinned={vi.fn()} />);

    expect(screen.getByRole("radio", { name: "记录交易" })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "复盘总结" })).toBeInTheDocument();
    expect(screen.getByText("至少先记录一条交易事件")).toBeInTheDocument();
  });

  it("does not switch to review tab when there are no trade events", () => {
    const journal = createJournal({
      id: "journal-no-events",
      symbol: "NQ",
      status: "open",
      events: [],
      images: [],
    });

    render(<JournalWorkspace journal={journal} store={createStore(journal)} isPinned={false} onTogglePinned={vi.fn()} />);

    fireEvent.click(screen.getByRole("radio", { name: "复盘总结" }));

    expect(screen.queryByRole("textbox", { name: "复盘" })).not.toBeInTheDocument();
  });

  it("switches to review tab only after there is a trade event and net position is flat", async () => {
    const journal = createJournal({
      id: "journal-2",
      symbol: "NQ",
      status: "open",
      events: [
        {
          id: "event-1",
          createdAt: "2026-04-26T10:00:00.000Z",
          updatedAt: "2026-04-26T10:00:00.000Z",
          timestamp: "2026-04-26T10:00:00.000Z",
          type: "log",
          content: "观察到回踩承接",
        },
      ],
      images: [],
    });

    render(<JournalWorkspace journal={journal} store={createStore(journal)} isPinned={false} onTogglePinned={vi.fn()} />);

    fireEvent.click(screen.getByRole("radio", { name: "复盘总结" }));

    expect(await screen.findByRole("textbox", { name: "复盘" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "返回记录" }));

    expect(screen.getByText("快速记录")).toBeInTheDocument();
  });

  it("completes review from the second step", async () => {
    const journal = createJournal({
      id: "journal-3",
      symbol: "NQ",
      status: "open",
      events: [
        {
          id: "event-1",
          createdAt: "2026-04-26T10:00:00.000Z",
          updatedAt: "2026-04-26T10:00:00.000Z",
          timestamp: "2026-04-26T10:00:00.000Z",
          type: "log",
          content: "计划执行完成",
        },
      ],
      images: [],
    });
    const store = createStore(journal);

    render(<JournalWorkspace journal={journal} store={store} isPinned={false} onTogglePinned={vi.fn()} />);

    fireEvent.click(screen.getByRole("radio", { name: "复盘总结" }));
    fireEvent.change(await screen.findByRole("textbox", { name: "复盘" }), {
      target: { value: "纪律执行到位" },
    });
    fireEvent.click(screen.getByRole("button", { name: "完成复盘" }));

    await waitFor(() => {
      expect(store.actions.finishJournal).toHaveBeenCalledWith("journal-3", "纪律执行到位", []);
    });
  });
});

function createStore(journal: TradeJournal) {
  return {
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
}
