import React from "react";
import { ConfigProvider } from "antd";
import { beforeEach, describe, expect, it, vi } from "vitest";

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

  it("wraps the app with the root Ant Design theme provider", async () => {
    await import("../main");

    expect(createRootMock).toHaveBeenCalledWith(document.getElementById("root"));
    expect(renderMock).toHaveBeenCalledTimes(1);

    const renderedTree = renderMock.mock.calls[0]?.[0] as React.ReactElement<{
      children: React.ReactNode;
    }>;
    expect(renderedTree.type).toBe(React.StrictMode);

    const providerElement = React.Children.only(renderedTree.props.children) as React.ReactElement<{
      theme: {
        token: {
          colorPrimary: string;
          fontFamily: string;
        };
      };
    }>;
    expect(providerElement.type).toBe(ConfigProvider);
    expect(providerElement.props.theme).toMatchObject({
      token: {
        colorPrimary: "#2563eb",
        fontFamily: '"SF Pro Text", "SF Pro Display", "Helvetica Neue", "Segoe UI", sans-serif',
      },
    });
    expect(providerElement.props.theme.token).not.toHaveProperty("borderRadius");
  });
});
