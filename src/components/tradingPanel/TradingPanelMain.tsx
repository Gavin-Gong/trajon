import {
  CalendarOutlined,
  DashboardOutlined,
  PlusOutlined,
  SettingOutlined,
} from "@ant-design/icons";
import {
  Badge as AntBadge,
  Button,
  Calendar,
  Card,
  Descriptions,
  Divider,
  Empty,
  Flex,
  FloatButton,
  Layout,
  Menu,
  Select,
  Space,
  Switch,
  Tag,
  Typography,
} from "antd";
import type { BadgeProps as AntBadgeProps, CalendarProps } from "antd";
import dayjs, { type Dayjs } from "dayjs";
import "dayjs/locale/zh-cn";
import { useEffect, useMemo, useState } from "react";

import {
  defaultCalendarSelection,
  groupJournalsByDay,
  netPosition,
  type TradeJournal,
} from "../../tradingPanelDomain";
import { openPanelWindow } from "../../panelWindow";
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
  type TradingPanelStore,
} from "./TradingPanelShared";
import type { AppSection } from "./TradingPanelTypes";

dayjs.locale("zh-cn");

const { Paragraph, Text, Title } = Typography;

export function MainSurface({ store }: { store: TradingPanelStore }) {
  const [section, setSection] = useState<AppSection>("analytics");
  const menuItems = [
    {
      key: "analytics",
      icon: <DashboardOutlined />,
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

  return (
    <main>
      <Layout hasSider style={{ minHeight: "100vh", background: "transparent" }}>
        <Layout.Sider
          width={280}
          theme="light"
          style={{
            background: "transparent",
          }}
        >
          <Card style={{ ...shellCardStyle, position: "sticky", top: 24 }}>
            <Flex vertical gap={24}>
              <Space orientation="vertical" size={12}>
                <Title
                  level={4}
                  style={{
                    margin: 0,
                    color: "var(--app-text)",
                    fontSize: "clamp(1.9rem,2vw,2.6rem)",
                    letterSpacing: "-0.04em",
                  }}
                >
                  Trading Flow
                </Title>
              </Space>

              <Menu
                mode="inline"
                selectedKeys={[section]}
                items={menuItems}
                onClick={({ key }) => setSection(key as AppSection)}
                style={{ borderInlineEnd: 0, background: "transparent" }}
              />
            </Flex>
          </Card>
        </Layout.Sider>

        <Layout.Content style={{ minWidth: 0 }}>
          {section === "analytics" ? <AnalyticsPanel store={store} /> : null}
          {section === "calendar" ? <CalendarHistorySection store={store} /> : null}
          {section === "settings" ? <SettingsSection store={store} /> : null}
        </Layout.Content>
      </Layout>
      <FloatButton
        type="primary"
        icon={<PlusOutlined />}
        tooltip={<span>打开浮动面板</span>}
        onClick={() => {
          void openPanelWindow();
        }}
      />
    </main>
  );
}

function AnalyticsPanel({ store }: { store: TradingPanelStore }) {
  void store;
  return (
    <Card style={shellCardStyle}>
      <Empty
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        description={
          <Space orientation="vertical" size={4}>
            <Text strong style={{ color: "var(--app-text)" }}>分析页面暂时留空</Text>
            <Text style={{ color: "var(--app-muted)" }}>后续会在这里逐步补回分析模块。</Text>
          </Space>
        }
      />
    </Card>
  );
}

function CalendarHistorySection({ store }: { store: TradingPanelStore }) {
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
                        ? "var(--app-accent)"
                        : "var(--app-text-soft)",
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
        <Flex justify="space-between" align="end" wrap="wrap" gap={16} style={{ marginBottom: 16 }}>
          <Space orientation="vertical" size={4}>
            <Title level={2} style={{ margin: 0, color: "var(--app-text)", letterSpacing: "-0.02em" }}>
              日历回看
            </Title>
            <Paragraph style={{ margin: 0, color: "var(--app-muted)" }}>
              按创建日期浏览历史 journal，点击查看细节链路。
            </Paragraph>
          </Space>
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
              <Text strong style={{ color: "var(--app-text)" }}>Journal 详情</Text>
              <Text style={{ color: "var(--app-muted)" }}>
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
              <Title level={3} style={{ margin: 0, color: "var(--app-text)" }}>
                {journal.symbol || "未命名"}
              </Title>
              <Tag color={tagColorForVariant(journal.status === "open" ? "accent" : "neutral")}>
                {journal.status === "open" ? "进行中" : "已完成"}
              </Tag>
            </Flex>
            <Text style={{ color: "var(--app-muted)" }}>
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
          <Paragraph style={{ margin: 0, color: "var(--app-muted)", lineHeight: 1.8 }}>
            {journal.summary}
          </Paragraph>
        ) : null}

        <Divider style={{ margin: 0 }} />

        <TimelineSection journal={journal} title="Journal 时间线" />
      </Space>
    </Card>
  );
}

function SettingsSection({ store }: { store: TradingPanelStore }) {
  const { config } = store.state;

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
          <div>
            <Title level={3} style={{ margin: 0, color: "var(--app-text)", letterSpacing: "-0.02em" }}>
              设置
            </Title>
            <Paragraph style={{ margin: "4px 0 0", color: "var(--app-muted)" }}>
              提醒频率保存在本地，用来支持浮动面板的盘中记录节奏。
            </Paragraph>
          </div>

          <Card style={mutedShellCardStyle}>
            <Flex justify="space-between" align="center" gap={16}>
              <Space orientation="vertical" size={4}>
                <Text strong style={{ color: "var(--app-text)" }}>启用提醒</Text>
                <Text style={{ color: "var(--app-muted)" }}>按固定频率提醒自己记录盘中观察。</Text>
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
              <Text strong style={{ color: "var(--app-text)" }}>提醒间隔</Text>
              <Paragraph style={{ margin: 0, color: "var(--app-muted)" }}>
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
        </Space>
      </Card>

      <Card style={{ ...shellCardStyle, height: "fit-content" }}>
        <Space orientation="vertical" size={16} style={{ width: "100%" }}>
          <div>
            <Title level={3} style={{ margin: 0, color: "var(--app-text)", letterSpacing: "-0.02em" }}>
              数据与导出
            </Title>
            <Paragraph style={{ margin: "4px 0 0", color: "var(--app-muted)" }}>
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
            <Text style={{ color: "var(--app-muted)", lineHeight: 1.8 }}>
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
