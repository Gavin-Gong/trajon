import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { createJournal } from "../../domain/journal";
import { QuickCaptureCard } from "../QuickCaptureCard";

describe("QuickCaptureCard", () => {
  it("renders event options with icon-backed accessible labels", () => {
    render(
      <QuickCaptureCard
        journal={createJournal({
          id: "journal-1",
          symbol: "BTCUSDT",
          status: "open",
          events: [],
        })}
        onRecord={vi.fn()}
      />,
    );

    expect(screen.getByRole("radio", { name: "开仓" })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "计划" })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "笔记" })).toBeInTheDocument();
  });

  it("shows add and reduce options after an entry exists", () => {
    render(
      <QuickCaptureCard
        journal={createJournal({
          id: "journal-1",
          symbol: "BTCUSDT",
          status: "open",
          events: [
            {
              id: "event-1",
              createdAt: "2026-04-25T09:30:00.000Z",
              updatedAt: "2026-04-25T09:30:00.000Z",
              timestamp: "2026-04-25T09:30:00.000Z",
              type: "entry",
              quantity: 1,
              price: 12.34,
              content: "首次进场",
            },
          ],
        })}
        onRecord={vi.fn()}
      />,
    );

    expect(screen.getByRole("radio", { name: "加仓" })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "减仓" })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "平仓" })).toBeInTheDocument();
  });

  it("renders numeric fields as spinbuttons and submits numeric values", async () => {
    const onRecord = vi.fn().mockResolvedValue({
      ok: true,
      event: {
        id: "event-1",
        createdAt: "2026-04-25T09:30:00.000Z",
        updatedAt: "2026-04-25T09:30:00.000Z",
        timestamp: "2026-04-25T09:30:00.000Z",
        type: "entry",
        content: "测试",
      },
    });

    render(
      <QuickCaptureCard
        journal={createJournal({
          id: "journal-1",
          symbol: "BTCUSDT",
          status: "open",
          events: [],
        })}
        onRecord={onRecord}
      />,
    );

    fireEvent.click(screen.getByRole("radio", { name: "开仓" }));

    const [quantityInput, priceInput] = screen.getAllByRole("spinbutton");

    fireEvent.change(quantityInput, { target: { value: "1.23" } });
    fireEvent.change(priceInput, { target: { value: "12.34" } });
    fireEvent.change(screen.getByRole("textbox"), {
      target: { value: "测试" },
    });
    fireEvent.click(screen.getByRole("button", { name: /记\s*录/ }));

    await waitFor(() => {
      expect(onRecord).toHaveBeenCalledWith({
        type: "entry",
        quantityInput: "1.23",
        priceInput: "12.34",
        content: "测试",
      });
    });
  });
});
