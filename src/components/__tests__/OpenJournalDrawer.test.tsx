import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { createJournal } from "../../domain/journal";
import { OpenJournalDrawer } from "../OpenJournalDrawer";

describe("OpenJournalDrawer", () => {
  it("places the close button on the right side of the drawer header", () => {
    const openJournal = createJournal({
      id: "open-journal",
      symbol: "NQ",
      market: "CME",
      status: "open",
      events: [],
      images: [],
    });

    render(<OpenJournalDrawer open journals={[openJournal]} onClose={() => undefined} />);

    const headerTitle = document.querySelector(".ant-drawer-header-title");
    expect(headerTitle).not.toBeNull();
    expect(headerTitle?.lastElementChild).toHaveClass("ant-drawer-close");
  });
});
