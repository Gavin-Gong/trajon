import { beforeEach, describe, expect, it, vi } from "vitest";

const convertFileSrcMock = vi.hoisted(() => vi.fn((value: string) => `asset://${value}`));
const pathMock = vi.hoisted(() => ({
  homeDir: vi.fn(),
  join: vi.fn(),
  normalize: vi.fn(),
}));
const fsMock = vi.hoisted(() => ({
  exists: vi.fn(),
  mkdir: vi.fn(),
  remove: vi.fn(),
  writeFile: vi.fn(),
}));

vi.mock("@tauri-apps/api/core", () => ({
  convertFileSrc: convertFileSrcMock,
}));

vi.mock("@tauri-apps/api/path", () => pathMock);
vi.mock("@tauri-apps/plugin-fs", () => fsMock);

import {
  buildJournalImageFileName,
  persistJournalImages,
  removeJournalImageFile,
  resolveJournalImageSrc,
} from "../journalImages";

describe("journalImages", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    pathMock.homeDir.mockResolvedValue("/Users/zen");
    pathMock.join.mockImplementation(async (...parts: string[]) => parts.join("/"));
    pathMock.normalize.mockImplementation(async (value: string) => value);
    fsMock.exists.mockResolvedValue(true);
    window.__TAURI_INTERNALS__ = {};
  });

  it("builds filenames as journalid-timestamp-symbol", () => {
    expect(buildJournalImageFileName("journal-1", 1713399000000, "BTCUSDT", ".png")).toBe(
      "journal-1-1713399000000-btcusdt.png",
    );
  });

  it("persists selected files under ~/trajon/images", async () => {
    vi.spyOn(Date, "now").mockReturnValue(1713399000000);
    const file = new File([new Uint8Array([1, 2, 3])], "chart.png", { type: "image/png" });

    const images = await persistJournalImages(
      {
        id: "journal-1",
        createdAt: "2026-04-18T00:00:00.000Z",
        updatedAt: "2026-04-18T00:00:00.000Z",
        symbol: "BTCUSDT",
        market: "BINANCE",
        summary: "",
        status: "open",
        events: [],
        images: [],
      },
      [file],
    );

    expect(fsMock.mkdir).toHaveBeenCalledWith("/Users/zen/trajon/images", { recursive: true });
    expect(fsMock.writeFile).toHaveBeenCalledWith(
      "/Users/zen/trajon/images/journal-1-1713399000000-btcusdt.png",
      expect.any(Uint8Array),
    );
    expect(images[0]?.relativePath).toBe("trajon/images/journal-1-1713399000000-btcusdt.png");
  });

  it("resolves and removes stored files", async () => {
    const src = await resolveJournalImageSrc("trajon/images/journal-1-1713399000000-btcusdt.png");
    await removeJournalImageFile("trajon/images/journal-1-1713399000000-btcusdt.png");

    expect(src).toBe("asset:///Users/zen/trajon/images/journal-1-1713399000000-btcusdt.png");
    expect(fsMock.remove).toHaveBeenCalledWith(
      "/Users/zen/trajon/images/journal-1-1713399000000-btcusdt.png",
    );
  });
});
