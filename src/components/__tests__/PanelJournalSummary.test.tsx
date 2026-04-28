import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { createJournal } from "../../domain/journal";
import { PanelJournalSummary } from "../PanelJournalSummary";

describe("PanelJournalSummary", () => {
  it("shows an edit trigger for an existing symbol and updates the journal to a new symbol", async () => {
    const currentJournal = createJournal({
      id: "journal-current",
      symbol: "NQ",
      createdAt: "2026-04-20T09:30:00.000Z",
      updatedAt: "2026-04-20T09:30:00.000Z",
      status: "open",
    });
    const recentJournal = createJournal({
      id: "journal-recent",
      symbol: "BTCUSDT",
      createdAt: "2026-04-22T09:30:00.000Z",
      updatedAt: "2026-04-22T09:30:00.000Z",
      status: "done",
    });
    const olderJournal = createJournal({
      id: "journal-older",
      symbol: "ETHUSDT",
      createdAt: "2026-04-21T09:30:00.000Z",
      updatedAt: "2026-04-21T09:30:00.000Z",
      status: "done",
    });
    const onUpdateSymbol = vi.fn().mockResolvedValue(undefined);

    render(
      <PanelJournalSummary
        journal={currentJournal}
        journals={[currentJournal, olderJournal, recentJournal]}
        onUpdateSymbol={onUpdateSymbol}
      />,
    );

    expect(screen.getByText("NQ")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "编辑标的" }));

    const combobox = screen.getByRole("combobox", { name: "交易标的" });
    fireEvent.mouseDown(combobox);

    const options = await screen.findAllByRole("option");
    expect(options[0]).toHaveTextContent("BTCUSDT");
    expect(options[1]).toHaveTextContent("ETHUSDT");

    fireEvent.change(combobox, { target: { value: "SOLUSDT" } });
    fireEvent.keyDown(combobox, { key: "Enter" });

    expect(onUpdateSymbol).toHaveBeenCalledWith("journal-current", "SOLUSDT");
  });

  it("shows the symbol select immediately when the journal symbol is empty and prefills the most recent symbol", () => {
    const currentJournal = createJournal({
      id: "journal-current",
      symbol: "",
      createdAt: "2026-04-20T09:30:00.000Z",
      updatedAt: "2026-04-20T09:30:00.000Z",
      status: "open",
    });
    const recentJournal = createJournal({
      id: "journal-recent",
      symbol: "BTCUSDT",
      createdAt: "2026-04-22T09:30:00.000Z",
      updatedAt: "2026-04-22T09:30:00.000Z",
      status: "done",
    });
    const olderJournal = createJournal({
      id: "journal-older",
      symbol: "ETHUSDT",
      createdAt: "2026-04-21T09:30:00.000Z",
      updatedAt: "2026-04-21T09:30:00.000Z",
      status: "done",
    });

    render(
      <PanelJournalSummary
        journal={currentJournal}
        journals={[olderJournal, currentJournal, recentJournal]}
        onUpdateSymbol={vi.fn().mockResolvedValue(undefined)}
      />,
    );

    const combobox = screen.getByRole("combobox", { name: "交易标的" }) as HTMLInputElement;
    expect(combobox).toHaveValue("BTCUSDT");
  });
});
