import { describe, expect, it, vi } from "vitest";

import {
  readTopmostState,
  toggleTopmostState,
} from "./detailTopmost";

describe("readTopmostState", () => {
  it("reads the current window topmost state", async () => {
    const isAlwaysOnTop = vi.fn(async () => true);

    await expect(
      readTopmostState({
        getCurrentWindow: () => ({ isAlwaysOnTop, setAlwaysOnTop: vi.fn() }),
      }),
    ).resolves.toBe(true);
  });
});

describe("toggleTopmostState", () => {
  it("turns topmost on when it is currently off", async () => {
    const setAlwaysOnTop = vi.fn(async () => undefined);

    await expect(
      toggleTopmostState(false, {
        getCurrentWindow: () => ({
          isAlwaysOnTop: vi.fn(async () => false),
          setAlwaysOnTop,
        }),
      }),
    ).resolves.toBe(true);

    expect(setAlwaysOnTop).toHaveBeenCalledWith(true);
  });

  it("turns topmost off when it is currently on", async () => {
    const setAlwaysOnTop = vi.fn(async () => undefined);

    await expect(
      toggleTopmostState(true, {
        getCurrentWindow: () => ({
          isAlwaysOnTop: vi.fn(async () => true),
          setAlwaysOnTop,
        }),
      }),
    ).resolves.toBe(false);

    expect(setAlwaysOnTop).toHaveBeenCalledWith(false);
  });
});
