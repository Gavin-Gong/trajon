import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { Modal } from "antd";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { createJournal } from "../../domain/journal";
import { JournalDetailPanel } from "../JournalDetailPanel";

const pickJournalImageFilesMock = vi.hoisted(() => vi.fn());
const resolveJournalImageSrcMock = vi.hoisted(() => vi.fn());

vi.mock("../../platform/journalImages", () => ({
  isJournalImageManagementAvailable: () => true,
  pickJournalImageFiles: pickJournalImageFilesMock,
  resolveJournalImageSrc: resolveJournalImageSrcMock,
}));

vi.mock("../../platform/panelWindow", () => ({
  openPanelWindow: vi.fn(() => Promise.resolve()),
}));

describe("JournalDetailPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resolveJournalImageSrcMock.mockResolvedValue("asset://image-1");
  });

  it("adds images through the store and deletes immediately from the thumbnail action", async () => {
    const journal = createJournal({
      id: "journal-1",
      symbol: "BTCUSDT",
      images: [
        {
          id: "image-1",
          journalId: "journal-1",
          createdAt: "2026-04-18T00:10:00.000Z",
          updatedAt: "2026-04-18T00:10:00.000Z",
          fileName: "journal-1-1713399000000-btcusdt.png",
          relativePath: "trajon/images/journal-1-1713399000000-btcusdt.png",
          sortOrder: 0,
        },
      ],
    });
    const file = new File([new Uint8Array([1, 2, 3])], "chart.png", { type: "image/png" });
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
        recordEvent: vi.fn(),
        addJournalImages: vi.fn().mockResolvedValue(undefined),
        removeJournalImage: vi.fn().mockResolvedValue(undefined),
        deleteJournal: vi.fn().mockResolvedValue(undefined),
        updateJournalSymbol: vi.fn().mockResolvedValue(undefined),
        updateConfig: vi.fn(),
        upsertPairPreset: vi.fn(),
        removePairPreset: vi.fn(),
        setDefaultPresetSymbol: vi.fn(),
        setPanelPinned: vi.fn(),
        refreshState: vi.fn(),
      },
    };

    pickJournalImageFilesMock.mockResolvedValue([file]);
    render(<JournalDetailPanel journal={journal} store={store} />);

    fireEvent.click(screen.getByRole("button", { name: "上传图片" }));

    await waitFor(() => {
      expect(store.actions.addJournalImages).toHaveBeenCalledWith("journal-1", [file]);
    });

    fireEvent.click(screen.getByRole("button", { name: /删除图片/ }));

    await waitFor(() => {
      expect(store.actions.removeJournalImage).toHaveBeenCalledWith("journal-1", "image-1");
    });

    const deleteButton = screen.getByRole("button", { name: /删除图片/ });
    expect(deleteButton).toHaveStyle({
      background: "rgba(255, 77, 79, 0.12)",
      borderColor: "rgba(255, 77, 79, 0.18)",
      color: "rgb(211, 47, 47)",
    });
  });

  it("opens image preview without showing the file name in the preview modal title", async () => {
    const infoSpy = vi.spyOn(Modal, "info").mockImplementation(() => ({ destroy: vi.fn(), update: vi.fn() } as never));
    const journal = createJournal({
      id: "journal-1",
      symbol: "BTCUSDT",
      images: [
        {
          id: "image-1",
          journalId: "journal-1",
          createdAt: "2026-04-18T00:10:00.000Z",
          updatedAt: "2026-04-18T00:10:00.000Z",
          fileName: "chart-setup.png",
          relativePath: "trajon/images/chart-setup.png",
          sortOrder: 0,
        },
      ],
    });

    render(<JournalDetailPanel journal={journal} />);

    fireEvent.click(screen.getByRole("button", { name: /预览图片 chart-setup\.png/ }));

    await waitFor(() => {
      expect(infoSpy).toHaveBeenCalledTimes(1);
    });

    const modalConfig = infoSpy.mock.calls[0]?.[0];
    expect(modalConfig?.title).toBeNull();

    infoSpy.mockRestore();
  });
});
