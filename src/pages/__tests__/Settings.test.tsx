import "./setupDesktopTests";

import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { Desktop } from "../Desktop";
import { getMockedStoreState } from "./setupDesktopTests";
import { openAntdSelect, chooseAntdOption, resetSeedState, seedState, typeIntoAntdAutoComplete } from "./desktopTestHelpers";

const mockedStoreState = getMockedStoreState();

describe("Settings", () => {
  beforeEach(() => {
    resetSeedState();
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
    expect(screen.queryByRole("button", { name: "开始记录今天的交易" })).not.toBeInTheDocument();
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
    expect(screen.queryByRole("button", { name: "开始记录今天的交易" })).not.toBeInTheDocument();
  });
});
