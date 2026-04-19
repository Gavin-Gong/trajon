import {
  BarChartOutlined,
  CalendarOutlined,
  DashboardOutlined,
  EllipsisOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  SettingOutlined,
} from "@ant-design/icons";
import {
  Button,
  Dropdown,
  Flex,
  Layout,
  Menu,
  Space,
  Typography,
} from "antd";
import type { MenuProps } from "antd";
import "dayjs/locale/zh-cn";
import dayjs from "dayjs";
import { useState } from "react";

import { AnalysisSection } from "../components/AnalysisSection";
import { CalendarHistorySection } from "../components/CalendarHistorySection";
import { DashboardSection } from "../components/DashboardSection";
import { SettingsSection } from "../components/SettingsSection";
import { TrajonBrandMark } from "../components/TrajonBrandMark";
import type { AppSection } from "../components/AppTypes";
import type { TradingJournalStore } from "../components/JournalStoreTypes";
import { useWindowFullscreenOnResize } from "../hooks/useWindowFullscreenOnResize";
import { openPanelWindow } from "../platform/panelWindow";

dayjs.locale("zh-cn");

const { Text } = Typography;
const SIDER_WIDTH = 200;
const SIDER_COLLAPSED_WIDTH = 80;
const HEADER_HEIGHT = 60;

const SECTION_META: Record<AppSection, { title: string; description: string }> = {
  dashboard: {
    title: "Dashboard",
    description: "总览当前记录入口与常用操作。",
  },
  analysis: {
    title: "分析",
    description: "查看累计已实现盈亏与每日盈亏分布。",
  },
  calendar: {
    title: "日历回看",
    description: "按日期查看 journal 记录与时间线。",
  },
  settings: {
    title: "设置",
    description: "管理预设、提醒与本地数据。",
  },
};

export function Main({ store }: { store: TradingJournalStore }) {
  const [section, setSection] = useState<AppSection>("dashboard");
  const [collapsed, setCollapsed] = useState(false);
  const isFullscreen = useWindowFullscreenOnResize();
  const menuItems = [
    {
      key: "dashboard",
      icon: <DashboardOutlined />,
      label: "Dashboard",
    },
    {
      key: "analysis",
      icon: <BarChartOutlined />,
      label: "分析",
    },
    {
      key: "calendar",
      icon: <CalendarOutlined />,
      label: "日历",
    },
    {
      key: "settings",
      icon: <SettingOutlined />,
      label: "设置",
    },
  ];
  const activeSection = SECTION_META[section];
  const presetMenuItems: MenuProps["items"] = store.state.pairPresets.map((preset) => ({
    key: preset.symbol,
    label: `开始交易 ${preset.symbol}`,
  }));
  const siderOffset = collapsed ? SIDER_COLLAPSED_WIDTH : SIDER_WIDTH;

  return (
    <main>
      <Layout hasSider style={{ minHeight: "100vh", background: "transparent" }}>
        <Layout.Sider
          width={SIDER_WIDTH}
          collapsedWidth={SIDER_COLLAPSED_WIDTH}
          collapsed={collapsed}
          trigger={null}
          theme="light"
          style={{
            position: "fixed",
            insetBlock: 0,
            insetInlineStart: 0,
            zIndex: 30,
            overflow: "auto",
            background: "rgba(255, 255, 255, 0.94)",
            boxShadow: "1px 0 0 rgba(15, 23, 42, 0.08)",
          }}
        >
          <Flex
            vertical
            gap={16}
            style={{
              minHeight: "100%",
              paddingBottom: 12,
              paddingTop: isFullscreen ? 12 : 32,
            }}
          >
            {collapsed ? null : (
              <Flex align="center" gap={10} style={{ minHeight: 48, paddingInline: 16 }}>
                <TrajonBrandMark />
                <Space orientation="vertical" size={0}>
                  <Text strong style={{ color: "rgba(0, 0, 0, 0.88)", fontSize: 16 }}>
                    Trajon
                  </Text>
                  <Text style={{ color: "rgba(0, 0, 0, 0.45)", fontSize: 12 }}>
                    Trading Flow
                  </Text>
                </Space>
              </Flex>
            )}

            <Menu
              mode="inline"
              inlineCollapsed={collapsed}
              selectedKeys={[section]}
              items={menuItems}
              onClick={({ key }) => setSection(key as AppSection)}
              style={{ borderInlineEnd: 0, background: "transparent" }}
            />
          </Flex>
        </Layout.Sider>

        <Layout.Content
          style={{
            minWidth: 0,
            marginInlineStart: siderOffset,
            transition: "margin-inline-start 0.2s ease",
          }}
        >
          <Flex
            align="center"
            justify="space-between"
            gap={16}
            style={{
              position: "fixed",
              top: 0,
              left: siderOffset,
              right: 0,
              height: HEADER_HEIGHT,
              paddingInline: 20,
              zIndex: 20,
              background: "rgba(255, 255, 255, 0.9)",
              backdropFilter: "blur(12px)",
              borderBottom: "1px solid rgba(15, 23, 42, 0.08)",
              transition: "left 0.2s ease",
            }}
          >
            <Flex align="center" gap={12} style={{ minWidth: 0 }}>
              <Button
                type="text"
                aria-label={collapsed ? "展开导航" : "收起导航"}
                icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
                onClick={() => setCollapsed((current) => !current)}
                style={{
                  width: 40,
                  height: 40,
                  fontSize: 20,
                  display: "inline-flex",
                  alignItems: "center",
                  justifyContent: "center",
                }}
              />

              <Space orientation="vertical" size={0} style={{ minWidth: 0 }}>
                <Text strong style={{ color: "rgba(0, 0, 0, 0.88)", fontSize: 17, lineHeight: 1.2 }}>
                  {activeSection.title}
                </Text>
                <Text style={{ color: "rgba(0, 0, 0, 0.45)", fontSize: 13, lineHeight: 1.3 }}>
                  {activeSection.description}
                </Text>
              </Space>
            </Flex>

            <Space.Compact>
              <Button
                type="primary"
                onClick={() => {
                  void openPanelWindow();
                }}
              >
                开始交易
              </Button>
              <Dropdown
                trigger={["click"]}
                menu={{
                  items:
                    presetMenuItems.length > 0
                      ? presetMenuItems
                      : [{ key: "empty", label: "暂无预设", disabled: true }],
                  onClick: ({ key }) => {
                    if (key === "empty") {
                      return;
                    }

                    void openPanelWindow({ presetSymbol: String(key) });
                  },
                }}
                placement="bottomRight"
              >
                <Button aria-label="更多交易对" disabled={presetMenuItems.length === 0} icon={<EllipsisOutlined />} />
              </Dropdown>
            </Space.Compact>
          </Flex>

          <div
            style={{
              padding: `${HEADER_HEIGHT + 16}px 20px 20px`,
            }}
          >
            {section === "dashboard" ? <DashboardSection store={store} /> : null}
            {section === "analysis" ? <AnalysisSection store={store} /> : null}
            {section === "calendar" ? <CalendarHistorySection store={store} /> : null}
            {section === "settings" ? <SettingsSection store={store} /> : null}
          </div>
        </Layout.Content>
      </Layout>
    </main>
  );
}
