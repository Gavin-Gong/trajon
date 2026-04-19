import { describe, expect, it } from "vitest";
import { createAppThemeConfig, fontFamily } from "./useAppTheme";

describe("createAppThemeConfig", () => {
  it("returns the shadcn-like theme tokens and semantic class hooks", () => {
    const config = createAppThemeConfig({
      buttonPrimary: "button-primary",
      buttonDefault: "button-default",
      buttonDanger: "button-danger",
      inputRoot: "input-root",
      inputElement: "input-element",
      inputError: "input-error",
      selectRoot: "select-root",
      selectPopup: "select-popup",
    });

    expect(config.theme).toMatchObject({
      token: {
        colorPrimary: "#262626",
        colorBgLayout: "#fafafa",
        borderRadius: 10,
        fontFamily,
      },
      components: {
        Button: {
          defaultBg: "#ffffff",
        },
        Input: {
          activeBorderColor: "#18181b",
        },
      },
    });

    expect(config.button?.classNames).toBeDefined();
    expect(config.input?.classNames).toBeDefined();
    expect(config.select?.classNames).toBeDefined();
  });
});
