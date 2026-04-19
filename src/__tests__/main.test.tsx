import React from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AppThemeProvider } from "../theme/AppThemeProvider";

const { createRootMock, renderMock } = vi.hoisted(() => {
  const renderMock = vi.fn();
  const createRootMock = vi.fn(() => ({ render: renderMock }));

  return { createRootMock, renderMock };
});

vi.mock("react-dom/client", () => ({
  createRoot: createRootMock,
  default: {
    createRoot: createRootMock,
  },
}));

vi.mock("../App", () => ({
  default: function MockApp() {
    return <div data-testid="mock-app" />;
  },
}));

describe("main entry", () => {
  beforeEach(() => {
    document.body.innerHTML = '<div id="root"></div>';
    createRootMock.mockClear();
    renderMock.mockClear();
  });

  it("wraps the app with the root theme provider", async () => {
    await import("../main");

    expect(createRootMock).toHaveBeenCalledWith(document.getElementById("root"));
    expect(renderMock).toHaveBeenCalledTimes(1);

    const renderedTree = renderMock.mock.calls[0]?.[0] as React.ReactElement<{
      children: React.ReactNode;
    }>;
    expect(renderedTree.type).toBe(React.StrictMode);

    const themeProviderElement = React.Children.only(renderedTree.props.children) as React.ReactElement<{
      children: React.ReactNode;
    }>;
    expect(themeProviderElement.type).toBe(AppThemeProvider);
  });
});
