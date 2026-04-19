import type { PropsWithChildren } from "react";
import { ConfigProvider } from "antd";
import { useAppTheme } from "./useAppTheme";

export function AppThemeProvider({ children }: PropsWithChildren) {
  const config = useAppTheme();

  return <ConfigProvider {...config}>{children}</ConfigProvider>;
}
