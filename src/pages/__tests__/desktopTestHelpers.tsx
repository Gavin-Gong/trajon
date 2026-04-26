import { fireEvent, screen } from "@testing-library/react";
import { expect } from "vitest";

import type { TradingPanelState } from "../../model/storage";
import { composeTradeEvent, createJournal } from "../../domain/journal";
import { getMockedStoreState } from "./setupDesktopTests";

const mockedStoreState = getMockedStoreState();

export function buildState(): TradingPanelState {
  const openJournal = createJournal({
    id: "open-journal",
    symbol: "NQ",
    createdAt: "2026-04-04T09:30:00.000Z",
    status: "open",
    summary: "等待二次确认后继续执行。",
  });
  const doneJournal = createJournal({
    id: "done-journal",
    symbol: "ES",
    createdAt: "2026-04-02T09:30:00.000Z",
    status: "done",
    summary: "纪律不错，但出场可以更果断。",
  });

  const entry = composeTradeEvent(
    {
      type: "entry",
      content: "Opening drive long",
      quantityInput: "2",
      priceInput: "18250",
    },
    0,
    "open",
  );
  const log = composeTradeEvent(
    {
      type: "log",
      content: "保持耐心，等待回踩确认。",
      quantityInput: "",
      priceInput: "",
    },
    2,
    "open",
  );

  expect(entry.ok).toBe(true);
  expect(log.ok).toBe(true);

  if (entry.ok && log.ok) {
    openJournal.events = [entry.event, log.event];
  }

  return {
    journals: [doneJournal, openJournal],
    config: {
      timerEnabled: true,
      timerInterval: 300,
    },
    pairPresets: [],
    defaultPresetSymbol: undefined,
    panelPinned: true,
  };
}

export function seedState(state: TradingPanelState) {
  mockedStoreState.value = JSON.parse(JSON.stringify(state)) as TradingPanelState;
}

export function resetSeedState() {
  mockedStoreState.value = buildState();
}

export function expectTextPresent(text: string | RegExp) {
  const matches = screen.getAllByText(text);
  expect(matches.length).toBeGreaterThanOrEqual(1);
}

export function openAntdSelect(testId: string) {
  const root = screen.getByTestId(testId);
  const trigger = root.querySelector('[role="combobox"]');
  expect(trigger).not.toBeNull();
  fireEvent.mouseDown(trigger as Element);
}

export function chooseAntdOption(name: string) {
  fireEvent.click(screen.getByRole("option", { name }));
}

export function typeIntoAntdAutoComplete(testId: string, value: string) {
  const root = screen.getByTestId(testId);
  const input = root.matches("input") ? root : root.querySelector("input");
  expect(input).not.toBeNull();
  fireEvent.change(input as Element, { target: { value } });
}
