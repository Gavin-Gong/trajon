import React from "react";
import ReactDOM from "react-dom/client";
import { ConfigProvider } from "antd";
import App from "./App";

const appTheme = {
  token: {
    colorPrimary: "#2563eb",
    borderRadius: 16,
    fontFamily: '"SF Pro Text", "SF Pro Display", "Helvetica Neue", "Segoe UI", sans-serif',
  },
};

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <ConfigProvider theme={appTheme}>
      <App />
    </ConfigProvider>
  </React.StrictMode>,
);
