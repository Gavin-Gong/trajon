import "./setupDesktopTests";

import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { message, Modal } from "antd";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Desktop } from "../Desktop";
import { getMockedRefreshState, getMockedStoreState } from "./setupDesktopTests";
import { openAntdSelect, chooseAntdOption, resetSeedState, seedState, typeIntoAntdAutoComplete } from "./desktopTestHelpers";

const mockedStoreState = getMockedStoreState();
const mockedRefreshState = getMockedRefreshState();
const dataBackupMock = vi.hoisted(() => ({
  buildDatabaseBackupFileName: vi.fn(() => "trajon-backup-20260426-090807.db"),
  createDatabaseBackup: vi.fn(() => Promise.resolve({
    fileName: "trajon-backup-20260426-090807.db",
    relativePath: "databases/backups/trajon-backup-20260426-090807.db",
  })),
  isDatabaseBackupAvailable: vi.fn(() => true),
  listDatabaseBackups: vi.fn(() => Promise.resolve([])),
  restoreDatabaseBackup: vi.fn(() => Promise.resolve()),
}));

vi.mock("../../platform/dataBackup", () => dataBackupMock);

describe("Settings", () => {
  beforeEach(() => {
    resetSeedState();
    mockedRefreshState.mockClear();
    dataBackupMock.buildDatabaseBackupFileName.mockReturnValue("trajon-backup-20260426-090807.db");
    dataBackupMock.createDatabaseBackup.mockResolvedValue({
      fileName: "trajon-backup-20260426-090807.db",
      relativePath: "databases/backups/trajon-backup-20260426-090807.db",
    });
    dataBackupMock.isDatabaseBackupAvailable.mockReturnValue(true);
    dataBackupMock.listDatabaseBackups.mockResolvedValue([]);
    dataBackupMock.restoreDatabaseBackup.mockResolvedValue(undefined);
  });

  it("renders trading pair preset controls in settings", () => {
    seedState({
      journals: [],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [{ symbol: "BTCUSDT", timeframe: "5m" }],
      defaultPresetSymbol: "BTCUSDT",
      panelPinned: true,
    });

    render(<Desktop view={{ view: "main" }} />);
    fireEvent.click(screen.getByRole("menuitem", { name: /设置/ }));

    expect(screen.getByText("交易对预设")).toBeInTheDocument();
    expect(screen.getByText("BTCUSDT")).toBeInTheDocument();
    expect(screen.getAllByText("5m").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText(/默认预设/).length).toBeGreaterThanOrEqual(1);
  });

  it("saves a typed trading pair preset with timeframe from settings", () => {
    seedState({
      journals: [],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [],
      defaultPresetSymbol: undefined,
      panelPinned: true,
    });

    render(<Desktop view={{ view: "main" }} />);
    fireEvent.click(screen.getByRole("menuitem", { name: /设置/ }));
    typeIntoAntdAutoComplete("pair-preset-symbol-input", "ethusdt");
    openAntdSelect("pair-preset-timeframe-select");
    chooseAntdOption("15m");
    fireEvent.click(screen.getByRole("button", { name: "保存为预设" }));

    const presetList = screen.getByTestId("pair-preset-list");
    expect(within(presetList).getByText("ETHUSDT")).toBeInTheDocument();
    expect(within(presetList).getAllByText("15m").length).toBeGreaterThanOrEqual(1);
  });

  it("updates an existing trading pair preset timeframe from settings", () => {
    seedState({
      journals: [],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [{ symbol: "BTCUSDT", timeframe: "5m" }],
      defaultPresetSymbol: undefined,
      panelPinned: true,
    });

    render(<Desktop view={{ view: "main" }} />);
    fireEvent.click(screen.getByRole("menuitem", { name: /设置/ }));
    typeIntoAntdAutoComplete("pair-preset-symbol-input", "btcusdt");
    openAntdSelect("pair-preset-timeframe-select");
    chooseAntdOption("15m");
    fireEvent.click(screen.getByRole("button", { name: "保存为预设" }));

    const presetList = screen.getByTestId("pair-preset-list");
    expect(within(presetList).getAllByText("BTCUSDT")).toHaveLength(1);
    expect(within(presetList).getAllByText("15m").length).toBeGreaterThanOrEqual(1);
  });

  it("prefills the default trading pair preset when opening panel without an active journal", () => {
    seedState({
      journals: [],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [{ symbol: "BTCUSDT", timeframe: "5m" }],
      defaultPresetSymbol: "BTCUSDT",
      panelPinned: false,
    });

    render(<Desktop view={{ view: "panel" }} />);
    expect(screen.getByText("BTCUSDT")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "开始记录" })).not.toBeInTheDocument();
  });

  it("lets the user set a saved preset as default from settings", async () => {
    seedState({
      journals: [],
      config: { timerEnabled: false, timerInterval: 300 },
      pairPresets: [
        { symbol: "BTCUSDT", timeframe: "5m" },
        { symbol: "ETHUSDT", timeframe: "15m" },
      ],
      defaultPresetSymbol: "BTCUSDT",
      panelPinned: false,
    });

    const { unmount } = render(<Desktop view={{ view: "main" }} />);
    fireEvent.click(screen.getByRole("menuitem", { name: /设置/ }));
    fireEvent.click(screen.getByRole("button", { name: "设为默认-ETHUSDT" }));

    expect(within(screen.getByTestId("pair-preset-list")).getAllByText(/默认预设/).length).toBeGreaterThanOrEqual(1);

    unmount();
    mockedStoreState.value = {
      ...mockedStoreState.value!,
      defaultPresetSymbol: "ETHUSDT",
    };
    render(<Desktop view={{ view: "panel" }} />);

    await expect(screen.findByText("ETHUSDT")).resolves.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "开始记录" })).not.toBeInTheDocument();
  });

  it("renders active backup controls in settings", () => {
    render(<Desktop view={{ view: "main" }} />);
    fireEvent.click(screen.getByRole("menuitem", { name: /设置/ }));

    expect(screen.getByText("数据库备份与恢复")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "创建备份" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "恢复备份" })).toBeDisabled();
    expect(screen.queryByText("导出 CSV（即将支持）")).not.toBeInTheDocument();
  });

  it("creates a database backup from settings", async () => {
    const successSpy = vi.spyOn(message, "success").mockImplementation(() => undefined as never);

    render(<Desktop view={{ view: "main" }} />);
    fireEvent.click(screen.getByRole("menuitem", { name: /设置/ }));
    fireEvent.click(screen.getByRole("button", { name: "创建备份" }));

    await expect(dataBackupMock.createDatabaseBackup).toHaveBeenCalledTimes(1);
    expect(successSpy).toHaveBeenCalledWith("已创建备份：trajon-backup-20260426-090807.db");
  });

  it("restores a selected backup and refreshes state", async () => {
    dataBackupMock.listDatabaseBackups.mockResolvedValue([
      {
        fileName: "trajon-backup-20260426-090807.db",
        absolutePath: "/Users/zen/Library/Application Support/dev.trajon/databases/backups/trajon-backup-20260426-090807.db",
      },
    ] as never);
    const confirmSpy = vi.spyOn(Modal, "confirm").mockImplementation(({ onOk }) => {
      void onOk?.();
      return { destroy: vi.fn(), update: vi.fn() } as never;
    });
    const successSpy = vi.spyOn(message, "success").mockImplementation(() => undefined as never);

    render(<Desktop view={{ view: "main" }} />);
    fireEvent.click(screen.getByRole("menuitem", { name: /设置/ }));
    await waitFor(() => expect(screen.getByRole("button", { name: "恢复备份" })).toBeEnabled());
    fireEvent.click(screen.getByRole("button", { name: "恢复备份" }));

    expect(await screen.findByText("恢复数据库备份")).toBeInTheDocument();
    fireEvent.mouseDown(screen.getByLabelText("选择备份文件"));
    fireEvent.click(await screen.findByRole("option", { name: "trajon-backup-20260426-090807.db" }));
    fireEvent.click(screen.getByRole("button", { name: /确认恢复/ }));

    await waitFor(() => expect(dataBackupMock.restoreDatabaseBackup).toHaveBeenCalledWith("trajon-backup-20260426-090807.db"));
    await waitFor(() => expect(mockedRefreshState).toHaveBeenCalledTimes(1));
    expect(confirmSpy).toHaveBeenCalled();
    await waitFor(() => expect(successSpy).toHaveBeenCalledWith("已恢复备份：trajon-backup-20260426-090807.db"));
  });
});
