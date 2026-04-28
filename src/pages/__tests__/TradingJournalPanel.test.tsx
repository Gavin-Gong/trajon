import "./setupDesktopTests";

import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { message } from "antd";
import { describe, expect, it, vi } from "vitest";

import { Desktop } from "../Desktop";
import { openAntdSelect, seedState } from "./desktopTestHelpers";
import { closePanelWindow } from "../../platform/panelLifecycle";

function switchToReviewTab() {
  fireEvent.click(screen.getByRole("radio", { name: "复盘总结" }));
}

vi.mock("../../platform/journalImages", () => ({
  isJournalImageManagementAvailable: () => false,
  pickJournalImageFiles: vi.fn(),
  resolveJournalImageSrc: vi.fn(async (relativePath: string) => `asset://${relativePath}`),
}));

describe("TradingJournalPanel", () => {
  it("requires a symbol before entering the review step", () => {
    const warningSpy = vi.spyOn(message, "warning").mockImplementation(() => ({
      then: Promise.resolve.bind(Promise),
    }) as never);

    seedState({
      journals: [
        {
          id: "open-journal",
          createdAt: "2026-04-25T09:30:00.000Z",
          updatedAt: "2026-04-25T09:30:00.000Z",
          symbol: "",
          summary: "",
          tags: ["纪律", "减仓偏早"],
          status: "open",
          events: [],
          images: [],
        },
      ],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      panelPinned: true,
    });

    render(<Desktop view={{ view: "panel", journalId: "open-journal" }} />);

    expect(screen.getByTitle("当前保持置顶")).toBeInTheDocument();
    expect(screen.getByText("快速记录")).toBeInTheDocument();

    switchToReviewTab();

    expect(warningSpy).toHaveBeenCalledWith("请先选择或填写交易标的");
    expect(screen.queryByRole("textbox", { name: "复盘" })).not.toBeInTheDocument();

    warningSpy.mockRestore();
  });

  it("shows the journal as done after finishing review", async () => {
    const successSpy = vi.spyOn(message, "success").mockImplementation(() => ({
      then: Promise.resolve.bind(Promise),
    }) as never);
    const closePanelWindowMock = vi.mocked(closePanelWindow);

    seedState({
      journals: [
        {
          id: "open-journal",
          createdAt: "2026-04-25T09:30:00.000Z",
          updatedAt: "2026-04-25T09:30:00.000Z",
          symbol: "NQ",
          summary: "",
          tags: ["纪律", "减仓偏早"],
          status: "open",
          events: [
            {
              id: "event-1",
              createdAt: "2026-04-25T09:35:00.000Z",
              updatedAt: "2026-04-25T09:35:00.000Z",
              timestamp: "2026-04-25T09:35:00.000Z",
              type: "log",
              content: "开盘后先做观察记录",
            },
          ],
          images: [],
        },
      ],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      panelPinned: true,
    });

    const { unmount } = render(<Desktop view={{ view: "panel", journalId: "open-journal" }} />);

    switchToReviewTab();
    fireEvent.change(await screen.findByRole("textbox", { name: "复盘" }), {
      target: { value: "纪律执行到位" },
    });
    fireEvent.click(screen.getByRole("button", { name: "完成复盘" }));

    await waitFor(() => {
      expect(screen.getByText("已完成")).toBeInTheDocument();
    });
    expect(successSpy).toHaveBeenCalledWith("保存成功", 2);
    expect(screen.getAllByText("纪律执行到位").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("已完成")).toBeInTheDocument();

    unmount();

    successSpy.mockRestore();
    closePanelWindowMock.mockClear();
  });

  it("shows review tags and image count in the panel after finishing", async () => {
    const successSpy = vi.spyOn(message, "success").mockImplementation(() => ({
      then: Promise.resolve.bind(Promise),
    }) as never);

    seedState({
      journals: [
        {
          id: "open-journal",
          createdAt: "2026-04-25T09:30:00.000Z",
          updatedAt: "2026-04-25T09:30:00.000Z",
          symbol: "NQ",
          summary: "",
          tags: ["纪律", "减仓偏早"],
          status: "open",
          events: [
            {
              id: "event-1",
              createdAt: "2026-04-25T09:35:00.000Z",
              updatedAt: "2026-04-25T09:35:00.000Z",
              timestamp: "2026-04-25T09:35:00.000Z",
              type: "log",
              content: "平仓后补复盘图片",
            },
          ],
          images: [
            {
              id: "image-1",
              journalId: "open-journal",
              createdAt: "2026-04-25T10:00:00.000Z",
              updatedAt: "2026-04-25T10:00:00.000Z",
              fileName: "setup-1.png",
              relativePath: "trajon/images/setup-1.png",
              sortOrder: 0,
            },
            {
              id: "image-2",
              journalId: "open-journal",
              createdAt: "2026-04-25T10:01:00.000Z",
              updatedAt: "2026-04-25T10:01:00.000Z",
              fileName: "setup-2.png",
              relativePath: "trajon/images/setup-2.png",
              sortOrder: 1,
            },
            {
              id: "image-3",
              journalId: "open-journal",
              createdAt: "2026-04-25T10:02:00.000Z",
              updatedAt: "2026-04-25T10:02:00.000Z",
              fileName: "setup-3.png",
              relativePath: "trajon/images/setup-3.png",
              sortOrder: 2,
            },
            {
              id: "image-4",
              journalId: "open-journal",
              createdAt: "2026-04-25T10:03:00.000Z",
              updatedAt: "2026-04-25T10:03:00.000Z",
              fileName: "setup-4.png",
              relativePath: "trajon/images/setup-4.png",
              sortOrder: 3,
            },
          ],
        },
      ],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      panelPinned: true,
    });

    const { unmount } = render(<Desktop view={{ view: "panel", journalId: "open-journal" }} />);

    switchToReviewTab();
    fireEvent.change(await screen.findByRole("textbox", { name: "复盘" }), {
      target: { value: "按计划执行，止盈太保守。" },
    });
    fireEvent.click(screen.getByRole("button", { name: "完成复盘" }));

    await waitFor(() => {
      expect(screen.getAllByText("复盘标签").length).toBeGreaterThanOrEqual(1);
    });

    expect(screen.getAllByText("纪律").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("减仓偏早").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("复盘图片").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("+1")).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getAllByAltText("setup-1.png").length).toBeGreaterThanOrEqual(1);
    });
    expect(screen.getAllByText("按计划执行，止盈太保守。").length).toBeGreaterThanOrEqual(1);

    unmount();
    successSpy.mockRestore();
  });

  it("shows historical tags from done journals in the review step", async () => {
    seedState({
      journals: [
        {
          id: "done-journal",
          createdAt: "2026-04-24T09:30:00.000Z",
          updatedAt: "2026-04-24T10:15:00.000Z",
          symbol: "ES",
          summary: "有执行力，但止盈偏早。",
          tags: ["纪律", "趋势跟随"],
          status: "done",
          events: [
            {
              id: "event-1",
              createdAt: "2026-04-25T09:35:00.000Z",
              updatedAt: "2026-04-25T09:35:00.000Z",
              timestamp: "2026-04-25T09:35:00.000Z",
              type: "log",
              content: "等待从历史标签选择",
            },
          ],
          images: [],
        },
        {
          id: "open-journal",
          createdAt: "2026-04-25T09:30:00.000Z",
          updatedAt: "2026-04-25T09:30:00.000Z",
          symbol: "NQ",
          summary: "",
          tags: [],
          status: "open",
          events: [],
          images: [],
        },
        {
          id: "open-journal-2",
          createdAt: "2026-04-25T11:00:00.000Z",
          updatedAt: "2026-04-25T11:10:00.000Z",
          symbol: "YM",
          summary: "",
          tags: ["不要出现"],
          status: "open",
          events: [],
          images: [],
        },
      ],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      panelPinned: true,
    });

    render(<Desktop view={{ view: "panel", journalId: "open-journal" }} />);

    switchToReviewTab();
    openAntdSelect("review-tags-select");

    await waitFor(() => {
      expect(screen.getByRole("option", { name: "纪律" })).toBeInTheDocument();
    });

    expect(screen.getByRole("option", { name: "趋势跟随" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "不要出现" })).not.toBeInTheDocument();
  });
});
