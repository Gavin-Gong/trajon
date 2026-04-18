import {
  CalendarOutlined,
  DashboardOutlined,
  EllipsisOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  SettingOutlined,
} from "@ant-design/icons";
import {
  AutoComplete,
  Badge as AntBadge,
  Button,
  Calendar,
  Card,
  Descriptions,
  Divider,
  Dropdown,
  Empty,
  Flex,
  Layout,
  Menu,
  Select,
  Space,
  Switch,
  Tag,
  Typography,
} from "antd";
import type { BadgeProps as AntBadgeProps, CalendarProps, MenuProps } from "antd";
import dayjs, { type Dayjs } from "dayjs";
import "dayjs/locale/zh-cn";
import { useEffect, useMemo, useState } from "react";

import {
  defaultCalendarSelection,
  groupJournalsByDay,
  netPosition,
  type TradeJournal,
} from "../../domain/tradingJournalDomain";
import { useWindowFullscreenOnResize } from "../../hooks/useWindowFullscreenOnResize";
import { TIMEFRAME_PRESET_VALUES, type TimeframePreset } from "../../model/tradingPairPreset";
import { openPanelWindow } from "../../platform/panelWindow";
import {
  formatJournalDate,
  formatNumber,
  REMINDER_INTERVALS,
  dashedShellCardStyle,
  mutedShellCardStyle,
  panelCardStyle,
  shellCardStyle,
  tagColorForVariant,
  TimelineSection,
  type TradingJournalStore,
} from "../../components/tradingJournal/TradingJournalShared";
import type { AppSection } from "../../components/tradingJournal/TradingJournalTypes";

dayjs.locale("zh-cn");

const { Paragraph, Text, Title } = Typography;
const SIDER_WIDTH = 200;
const SIDER_COLLAPSED_WIDTH = 80;
const HEADER_HEIGHT = 60;

const SECTION_META: Record<AppSection, { title: string; description: string }> = {
  dashboard: {
    title: "Dashboard",
    description: "总览当前记录入口与常用操作。",
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

export function TradingJournalMain({ store }: { store: TradingJournalStore }) {
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
            {section === "dashboard" ? <DashboardPanel /> : null}
            {section === "calendar" ? <CalendarHistorySection store={store} /> : null}
            {section === "settings" ? <SettingsSection store={store} /> : null}
          </div>
        </Layout.Content>
      </Layout>
    </main>
  );
}

function TrajonBrandMark() {
  return (
    <svg
      data-testid="trading-journal-brand-mark"
      width="34"
      height="34"
      viewBox="0 0 256 256"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
    >
      <circle
        cx="152"
        cy="88"
        r="74"
        stroke="#F7C548"
        strokeWidth="30"
        strokeLinecap="round"
        strokeDasharray="330 140"
        transform="rotate(-26 152 88)"
      />
      <circle
        cx="104"
        cy="168"
        r="74"
        stroke="#34C2D8"
        strokeWidth="30"
        strokeLinecap="round"
        strokeDasharray="330 140"
        transform="rotate(154 104 168)"
      />
      <circle cx="136" cy="100" r="54" fill="#050505" />
      <circle cx="112" cy="156" r="54" fill="#050505" />
      <circle cx="136" cy="100" r="18" fill="#F7C548" />
      <circle cx="112" cy="156" r="18" fill="#34C2D8" />
    </svg>
  );
}

function DashboardPanel() {
  return (
    <Space orientation="vertical" size={20} style={{ width: "100%" }}>
      <Empty
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        description={
          <Space orientation="vertical" size={4}>
            <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>Dashboard 暂时留空</Text>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>后续会在这里逐步补回 dashboard 模块。</Text>
          </Space>
        }
      />
    </Space>
  );
}

function CalendarHistorySection({ store }: { store: TradingJournalStore }) {
  const journals = store.state.journals;
  const [displayedMonth, setDisplayedMonth] = useState<Dayjs>(() => dayjs());
  const [selectedJournalId, setSelectedJournalId] = useState<string | undefined>();
  const grouped = useMemo(() => groupJournalsByDay(journals), [journals]);
  const groupedMap = useMemo(
    () => new Map(grouped.map((group) => [group.dayKey, group.journals])),
    [grouped],
  );
  const monthJournals = useMemo(
    () =>
      journals
        .filter((journal) => {
          const created = dayjs(journal.createdAt);
          return (
            created.year() === displayedMonth.year() &&
            created.month() === displayedMonth.month()
          );
        })
        .sort(
          (left, right) =>
            new Date(left.createdAt).getTime() - new Date(right.createdAt).getTime(),
        ),
    [displayedMonth, journals],
  );

  useEffect(() => {
    const selection = defaultCalendarSelection(displayedMonth.toDate(), journals);

    setSelectedJournalId((currentSelection) => {
      if (!currentSelection) {
        return selection?.id;
      }

      const stillVisible = monthJournals.some((journal) => journal.id === currentSelection);
      return stillVisible ? currentSelection : selection?.id;
    });
  }, [displayedMonth, journals, monthJournals]);

  const selectedJournal = journals.find((journal) => journal.id === selectedJournalId);

  const journalBadgeStatus = (journal: TradeJournal): AntBadgeProps["status"] =>
    journal.status === "open" ? "processing" : "success";

  const dateCellRender = (current: Dayjs) => {
    const dayKey = current.format("YYYY-MM-DD");
    const dayJournals = groupedMap.get(dayKey) ?? [];

    if (dayJournals.length === 0) {
      return null;
    }

    return (
      <div style={{ margin: 0 }}>
        {dayJournals.map((journal) => (
          <div key={journal.id} style={{ marginBottom: 4 }}>
            <AntBadge
              status={journalBadgeStatus(journal)}
              text={
                <Button
                  type="link"
                  size="small"
                      style={{
                        fontWeight: selectedJournalId === journal.id ? 700 : 500,
                        color:
                          selectedJournalId === journal.id
                            ? "#2563eb"
                            : "rgba(0, 0, 0, 0.65)",
                      }}
                  onClick={(event) => {
                    event.stopPropagation();
                    setSelectedJournalId(journal.id);
                  }}
                >
                  {journal.symbol || "未命名"}
                </Button>
              }
            />
          </div>
        ))}
      </div>
    );
  };

  const cellRender: CalendarProps<Dayjs>["cellRender"] = (current, info) => {
    if (info.type === "date") {
      return dateCellRender(current);
    }

    return info.originNode;
  };

  return (
    <div
      style={{
        display: "grid",
        gap: 24,
        gridTemplateColumns: "minmax(0, 1fr) 380px",
      }}
    >
      <Card style={shellCardStyle}>
        <Flex justify="end" align="center" wrap="wrap" gap={16} style={{ marginBottom: 16 }}>
          <Tag color={tagColorForVariant("neutral")}>{monthJournals.length} 条</Tag>
        </Flex>
        <Calendar
          mode="month"
          value={displayedMonth}
          cellRender={cellRender}
          onPanelChange={(date) => setDisplayedMonth(date)}
          onSelect={(date) => setDisplayedMonth(date)}
        />
      </Card>

      <JournalDetailPanel journal={selectedJournal} />
    </div>
  );
}

function JournalDetailPanel({ journal }: { journal: TradeJournal | undefined }) {
  if (!journal) {
    return (
      <Card style={dashedShellCardStyle}>
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={
            <Space orientation="vertical" size={4}>
              <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>Journal 详情</Text>
              <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>
                从左侧日历选择一条 journal，这里会展示时间线与总结。
              </Text>
            </Space>
          }
        />
      </Card>
    );
  }

  return (
    <Card style={{ ...shellCardStyle, height: "fit-content" }}>
      <Space orientation="vertical" size={20} style={{ width: "100%" }}>
        <Flex justify="space-between" align="start" gap={12}>
          <Space orientation="vertical" size={8}>
            <Flex align="center" gap={8} wrap="wrap">
              <Title level={5} style={{ margin: 0, color: "rgba(0, 0, 0, 0.88)" }}>
                {journal.symbol || "未命名"}
              </Title>
              <Tag color={tagColorForVariant(journal.status === "open" ? "accent" : "neutral")}>
                {journal.status === "open" ? "进行中" : "已完成"}
              </Tag>
            </Flex>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>
              {journal.market || "未标记市场"} · {formatJournalDate(journal.createdAt)}
            </Text>
          </Space>
          <Tag color={tagColorForVariant("neutral")}>净仓 {formatNumber(netPosition(journal.events))}</Tag>
        </Flex>

        <Descriptions
          size="small"
          column={1}
          items={[
            { key: "events", label: "事件数量", children: `${journal.events.length} 条` },
            {
              key: "status",
              label: "状态",
              children: journal.status === "open" ? "进行中" : "已完成",
            },
          ]}
        />

        {journal.summary ? (
          <Paragraph style={{ margin: 0, color: "rgba(0, 0, 0, 0.45)", lineHeight: 1.8 }}>
            {journal.summary}
          </Paragraph>
        ) : null}

        <Divider style={{ margin: 0 }} />

        <TimelineSection journal={journal} title="Journal 时间线" />
      </Space>
    </Card>
  );
}

function SettingsSection({ store }: { store: TradingJournalStore }) {
  const { config } = store.state;
  const [presetSymbolInput, setPresetSymbolInput] = useState("");
  const [presetTimeframe, setPresetTimeframe] = useState<TimeframePreset>("5m");

  const presetOptions = store.state.pairPresets.map((preset) => ({
    value: preset.symbol,
    label: preset.symbol,
  }));

  const canSavePreset = presetSymbolInput.trim().length > 0;

  return (
    <div
      style={{
        display: "grid",
        gap: 24,
        gridTemplateColumns: "minmax(0, 1fr) 320px",
      }}
    >
      <Card style={shellCardStyle}>
        <Space orientation="vertical" size={16} style={{ width: "100%" }}>
          <Card style={mutedShellCardStyle}>
            <Flex justify="space-between" align="center" gap={16}>
              <Space orientation="vertical" size={4}>
                <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>启用提醒</Text>
                <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>按固定频率提醒自己记录盘中观察。</Text>
              </Space>
              <Switch
                checked={config.timerEnabled}
                onChange={(checked) => {
                  store.actions.updateConfig({ timerEnabled: checked });
                }}
              />
            </Flex>
          </Card>

          <Card style={panelCardStyle}>
            <Space orientation="vertical" size={12} style={{ width: "100%" }}>
              <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>提醒间隔</Text>
              <Paragraph style={{ margin: 0, color: "rgba(0, 0, 0, 0.45)" }}>
                建议与自己的盘中节奏保持一致，避免提醒过密造成打断。
              </Paragraph>
              <Select
                value={config.timerInterval}
                disabled={!config.timerEnabled}
                options={REMINDER_INTERVALS.map((preset) => ({
                  value: preset.value,
                  label: preset.label,
                }))}
                onChange={(value) => {
                  store.actions.updateConfig({ timerInterval: value });
                }}
              />
            </Space>
          </Card>

          <Card style={panelCardStyle}>
            <Space orientation="vertical" size={12} style={{ width: "100%" }}>
              <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>交易对预设</Text>
              <Paragraph style={{ margin: 0, color: "rgba(0, 0, 0, 0.45)" }}>
                为常用标的设置默认周期，并选择一个默认预设用于新建时自动带出。
              </Paragraph>

              <Flex gap={12} wrap="wrap" align="end">
                <Space orientation="vertical" size={8} style={{ minWidth: 220, flex: 1 }}>
                  <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>Symbol</Text>
                  <div data-testid="pair-preset-symbol-input">
                    <AutoComplete
                      value={presetSymbolInput}
                      virtual={false}
                      options={presetSymbolInput.trim().length === 0 ? [] : presetOptions}
                      placeholder="输入或选择交易对，例如 BTCUSDT"
                      filterOption={(inputValue: string, option?: { value?: string }) =>
                        (option?.value ?? "").toLowerCase().includes(inputValue.toLowerCase())
                      }
                      onChange={(value: string) => setPresetSymbolInput(value)}
                      style={{
                        width: "100%",
                      }}
                    />
                  </div>
                </Space>

                <Space orientation="vertical" size={8} style={{ minWidth: 140 }}>
                  <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>Timeframe</Text>
                  <div data-testid="pair-preset-timeframe-select">
                    <Select
                      value={presetTimeframe}
                      virtual={false}
                      options={TIMEFRAME_PRESET_VALUES.map((value) => ({ value, label: value }))}
                      onChange={(value) => setPresetTimeframe(value as TimeframePreset)}
                      style={{
                        minWidth: 140,
                      }}
                    />
                  </div>
                </Space>

                <Button
                  type="primary"
                  disabled={!canSavePreset}
                  onClick={() => {
                    if (!canSavePreset) {
                      return;
                    }

                    store.actions.upsertPairPreset(presetSymbolInput, presetTimeframe);
                    setPresetSymbolInput("");
                  }}
                >
                  保存为预设
                </Button>
              </Flex>

              <Button
                type="default"
                onClick={() => {
                  store.actions.upsertPairPreset("BTCUSDT", "5m");
                  store.actions.setDefaultPresetSymbol("BTCUSDT");
                }}
                style={{ display: "none" }}
              >
                测试入口
              </Button>

              <Space data-testid="pair-preset-list" orientation="vertical" size={8} style={{ width: "100%" }}>
                <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>默认预设</Text>
                {store.state.pairPresets.length === 0 ? (
                  <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>还没有保存任何交易对预设。</Text>
                ) : (
                  store.state.pairPresets.map((preset) => {
                     const isDefault = store.state.defaultPresetSymbol === preset.symbol;
                     return (
                        <Flex key={preset.symbol} justify="space-between" align="center" gap={12}>
                          <Space align="center" size={8}>
                            <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>{preset.symbol}</Text>
                            <Tag color={tagColorForVariant("neutral")}>{preset.timeframe}</Tag>
                            {isDefault ? <Tag color={tagColorForVariant("accent")}>默认预设</Tag> : null}
                          </Space>
                         {isDefault ? null : (
                           <Button
                             size="small"
                             onClick={() => {
                               store.actions.setDefaultPresetSymbol(preset.symbol);
                             }}
                             aria-label={`设为默认-${preset.symbol}`}
                           >
                             设为默认
                           </Button>
                         )}
                       </Flex>
                     );
                   })
                 )}
              </Space>
            </Space>
          </Card>
        </Space>
      </Card>

      <Card style={{ ...shellCardStyle, height: "fit-content" }}>
        <Space orientation="vertical" size={16} style={{ width: "100%" }}>
          <div>
            <Title level={5} style={{ margin: 0, color: "rgba(0, 0, 0, 0.88)", letterSpacing: "-0.02em" }}>
              数据与导出
            </Title>
            <Paragraph style={{ margin: "4px 0 0", color: "rgba(0, 0, 0, 0.45)" }}>
              目前数据保存在本地。导出入口保留在这里，便于后续接入 CSV。
            </Paragraph>
          </div>

          <Descriptions
            size="small"
            column={1}
            items={[
              { key: "storage", label: "数据位置", children: "本地存储" },
              { key: "export", label: "导出状态", children: "即将支持" },
            ]}
          />

          <Card style={dashedShellCardStyle}>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)", lineHeight: 1.8 }}>
              当前版本优先把记录链路和复盘体验打磨稳定，导出能力后续接入。
            </Text>
          </Card>

          <Button block onClick={() => undefined}>
            导出 CSV（即将支持）
          </Button>
        </Space>
      </Card>
    </div>
  );
}
