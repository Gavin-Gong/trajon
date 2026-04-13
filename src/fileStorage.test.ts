import { beforeEach, describe, expect, it, vi } from "vitest";
import { BaseDirectory, exists, mkdir, readTextFile, writeTextFile } from "@tauri-apps/plugin-fs";

const fsMock = vi.hoisted(() => ({
  BaseDirectory: {
    Home: "Home",
  },
  exists: vi.fn(),
  mkdir: vi.fn(),
  readTextFile: vi.fn(),
  writeTextFile: vi.fn(),
}));

vi.mock("@tauri-apps/plugin-fs", () => fsMock);

import { readDemoJson, writeDemoJson } from "./fileStorage";

declare global {
  interface Window {
    __TAURI_INTERNALS__?: unknown;
  }
}

describe("fileStorage", () => {
  beforeEach(() => {
    delete window.__TAURI_INTERNALS__;
    vi.mocked(exists).mockReset();
    vi.mocked(mkdir).mockReset();
    vi.mocked(readTextFile).mockReset();
    vi.mocked(writeTextFile).mockReset();
  });

  it("throws when Tauri fs is unavailable", async () => {
    await expect(readDemoJson()).rejects.toThrow(
      "Tauri file storage is only available in the desktop app",
    );
  });

  it("writes demo json to the home-scoped file", async () => {
    window.__TAURI_INTERNALS__ = {};

    await writeDemoJson({ hello: "world" });

    expect(mkdir).toHaveBeenCalledWith(".trading-flow", {
      baseDir: BaseDirectory.Home,
      recursive: true,
    });
    expect(writeTextFile).toHaveBeenCalledWith(
      ".trading-flow/demo.json",
      '{"hello":"world"}',
      { baseDir: BaseDirectory.Home },
    );
  });

  it("rejects null writes so missing files stay distinguishable", async () => {
    window.__TAURI_INTERNALS__ = {};

    await expect(writeDemoJson(null)).rejects.toThrow("Demo JSON value cannot be null");
    expect(writeTextFile).not.toHaveBeenCalled();
  });

  it("reads parsed demo json from the home-scoped file", async () => {
    window.__TAURI_INTERNALS__ = {};
    vi.mocked(exists).mockResolvedValue(true);
    vi.mocked(readTextFile).mockResolvedValue('{"hello":"world"}');

    await expect(readDemoJson()).resolves.toEqual({ hello: "world" });
    expect(readTextFile).toHaveBeenCalledWith(".trading-flow/demo.json", {
      baseDir: BaseDirectory.Home,
    });
  });

  it("returns null when the demo file does not exist", async () => {
    window.__TAURI_INTERNALS__ = {};
    vi.mocked(exists).mockResolvedValue(false);

    await expect(readDemoJson()).resolves.toBeNull();
  });

  it("returns null when the file disappears before read completes", async () => {
    window.__TAURI_INTERNALS__ = {};
    vi.mocked(exists).mockResolvedValue(true);
    vi.mocked(readTextFile).mockRejectedValue(new Error("No such file or directory"));

    await expect(readDemoJson()).resolves.toBeNull();
  });

  it("throws a clear error when demo json is invalid", async () => {
    window.__TAURI_INTERNALS__ = {};
    vi.mocked(exists).mockResolvedValue(true);
    vi.mocked(readTextFile).mockResolvedValue("not-json");

    await expect(readDemoJson()).rejects.toThrow("Invalid JSON in demo.json");
  });
});
