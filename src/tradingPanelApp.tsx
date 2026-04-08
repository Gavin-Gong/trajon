import { getCurrentWindow } from "@tauri-apps/api/window";
import {
  CalendarOutlined,
  LineChartOutlined,
  DashboardOutlined,
  PlusOutlined,
  PushpinFilled,
  PushpinOutlined,
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
  Form,
  Input,
  Layout,
  Menu,
  Modal,
  Select,
  Segmented,
  Space,
  Statistic,
  Switch,
  Tag,
  Typography,
} from "antd";
import type { BadgeProps as AntBadgeProps, CalendarProps } from "antd";
import dayjs, { type Dayjs } from "dayjs";
import "dayjs/locale/zh-cn";
import { type CSSProperties, useEffect, useMemo, useRef, useState } from "react";

import {
  availableEventTypes,
  composeTradeEvent,
  currentJournal as selectCurrentJournal,
  defaultCalendarSelection,
  groupJournalsByDay,
  isZero,
  netPosition,
  type TradeEvent,
  type TradeEventComposerError,
  type TradeEventDraft,
  type TradeEventType,
  type TradeJournal,
} from "./tradingPanelDomain";
import {
  TIMEFRAME_PRESET_VALUES,
  type TimeframePreset,
  type TradingPairPreset,
} from "./model/tradingPairPreset";

dayjs.locale("zh-cn");
import {
  openPanelWindow,
  PANEL_BOUNDS_STORAGE_KEY,
  serializePanelBounds,
} from "./panelWindow";
import { useTradingPanelState } from "./tradingPanelStore";

type AppView = "main" | "panel";
type AppSection = "analytics" | "chart" | "calendar" | "settings";
type AppTagVariant = "accent" | "danger" | "neutral" | "success" | "warning";

const CHART_SYMBOL_OPTIONS = [
  { label: "BTCUSDT", value: "BINANCE:BTCUSDT" },
  { label: "ETHUSDT", value: "BINANCE:ETHUSDT" },
  { label: "NQ", value: "CME_MINI:NQ1!" },
  { label: "ES", value: "CME_MINI:ES1!" },
] as const;

const CHART_INTERVAL_OPTIONS = ["1", "5", "15", "30", "60", "240", "1D"] as const;

const { Paragraph, Text, Title } = Typography;

const shellCardStyle: CSSProperties = {
  borderRadius: 28,
  borderColor: "var(--app-border)",
  background: "var(--app-surface-strong)",
  boxShadow: "var(--app-shadow-soft)",
};

const mutedShellCardStyle: CSSProperties = {
  ...shellCardStyle,
  background: "var(--app-panel-muted)",
  boxShadow: "none",
};

const dashedShellCardStyle: CSSProperties = {
  ...mutedShellCardStyle,
  background: "transparent",
  borderStyle: "dashed",
};

const panelCardStyle: CSSProperties = {
  ...shellCardStyle,
  borderRadius: 24,
};

export function TradingPanelDesktop({ view }: { view: AppView }) {
  const store = useTradingPanelState();

  if (view === "panel") {
    return <PanelSurface store={store} />;
  }

  return <MainSurface store={store} />;
}

function MainSurface({ store }: { store: ReturnType<typeof useTradingPanelState> }) {
  const [section, setSection] = useState<AppSection>("analytics");
  const menuItems = [
    {
      key: "analytics",
      icon: <DashboardOutlined />,
      label: "分析",
    },
    {
      key: "chart",
      icon: <LineChartOutlined />,
      label: "Chart",
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
            padding: 24,
            borderInlineEnd: "1px solid var(--app-border)",
          }}
        >
          <Card style={{ ...shellCardStyle, position: "sticky", top: 24 }}>
            <Flex vertical gap={24}>
              <Space orientation="vertical" size={12}>
                <Tag color={tagColorForVariant("accent")} style={{ width: "fit-content", marginInlineEnd: 0 }}>
                  Trading Flow
                </Tag>
                <Title level={1} style={{ margin: 0, color: "var(--app-text)", fontSize: "clamp(1.9rem,2vw,2.6rem)", letterSpacing: "-0.04em" }}>
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

        <Layout.Content style={{ minWidth: 0, padding: 24 }}>
          {section === "analytics" ? <AnalyticsPanel store={store} /> : null}
          {section === "chart" ? <ChartSection store={store} /> : null}
          {section === "calendar" ? <CalendarHistorySection store={store} /> : null}
          {section === "settings" ? <SettingsSection store={store} /> : null}
        </Layout.Content>
      </Layout>
      <FloatButton
        type="primary"
        icon={<PlusOutlined />}
        tooltip={<span>打开浮动面板</span>}
        onClick={() => void openPanelWindow()}
      />
    </main>
  );
}

function AnalyticsPanel({ store }: { store: ReturnType<typeof useTradingPanelState> }) {
  void store;
  return (
    <Card style={shellCardStyle}>
      <Empty
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        description={
          <Space orientation="vertical" size={4}>
            <Text strong style={{ color: "var(--app-text)" }}>
              分析页面暂时留空
            </Text>
            <Text style={{ color: "var(--app-muted)" }}>
              后续会在这里逐步补回分析模块。
            </Text>
          </Space>
        }
      />
    </Card>
  );
}

function ChartSection({ store }: { store: ReturnType<typeof useTradingPanelState> }) {
  const { symbol, interval } = store.state.chart;

  return (
    <Space orientation="vertical" size={24} style={{ width: "100%" }}>
      <Card style={shellCardStyle}>
        <Space orientation="vertical" size={8} style={{ width: "100%" }}>
          <Title level={2} style={{ margin: 0, color: "var(--app-text)", letterSpacing: "-0.02em" }}>
            图表工作台
          </Title>
          <Paragraph style={{ margin: 0, color: "var(--app-muted)" }}>
            这里将集成 TradingView widget，并提供标的与时间周期切换。
          </Paragraph>
        </Space>
      </Card>

      <Card style={shellCardStyle}>
        <Flex gap={16} wrap="wrap" align="end">
          <Space orientation="vertical" size={8} style={{ minWidth: 240 }}>
            <Text strong style={{ color: "var(--app-text)" }}>选择标的</Text>
            <select
              data-testid="chart-symbol-select"
              value={symbol}
              onChange={(event) =>
                store.actions.updateChartView({ symbol: event.target.value })
              }
              style={{
                minHeight: 36,
                borderRadius: 8,
                border: "1px solid var(--app-border)",
                padding: "0 12px",
                background: "white",
                color: "var(--app-text)",
              }}
            >
              {CHART_SYMBOL_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.value}
                </option>
              ))}
            </select>
          </Space>

          <Space orientation="vertical" size={8} style={{ minWidth: 180 }}>
            <Text strong style={{ color: "var(--app-text)" }}>时间周期</Text>
            <select
              data-testid="chart-interval-select"
              value={interval}
              onChange={(event) =>
                store.actions.updateChartView({ interval: event.target.value })
              }
              style={{
                minHeight: 36,
                borderRadius: 8,
                border: "1px solid var(--app-border)",
                padding: "0 12px",
                background: "white",
                color: "var(--app-text)",
              }}
            >
              {CHART_INTERVAL_OPTIONS.map((value) => (
                <option key={value} value={value}>
                  {value}
                </option>
              ))}
            </select>
          </Space>
        </Flex>
      </Card>

      <TradingViewWidgetChart symbol={symbol} interval={interval} />
    </Space>
  );
}

function TradingViewWidgetChart({
  symbol,
  interval,
}: {
  symbol: string;
  interval: string;
}) {
  const containerRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!containerRef.current) {
      return;
    }

    containerRef.current.dataset.symbol = symbol;
    containerRef.current.dataset.interval = interval;
    containerRef.current.replaceChildren();

    const script = document.createElement("script");
    script.type = "text/javascript";
    script.async = true;
    script.src = "https://s3.tradingview.com/external-embedding/embed-widget-advanced-chart.js";
    script.text = JSON.stringify({
      autosize: true,
      symbol,
      interval,
      timezone: "Asia/Shanghai",
      theme: "light",
      style: "1",
      locale: "zh_CN",
      allow_symbol_change: false,
      hide_top_toolbar: false,
      hide_legend: false,
      support_host: "https://www.tradingview.com",
    });

    containerRef.current.appendChild(script);
  }, [interval, symbol]);

  return (
    <Card style={shellCardStyle}>
      <Space orientation="vertical" size={12} style={{ width: "100%" }}>
        <Text style={{ color: "var(--app-muted)" }}>
          TradingView 图表将在这里加载，指标和划线功能将在图表内部提供。
        </Text>
        <Text data-testid="tradingview-widget-status" style={{ color: "var(--app-text)" }}>
          当前图表: {symbol} / {interval}
        </Text>
        <div data-testid="tradingview-widget-shell" style={{ width: "100%" }}>
          <div
            ref={containerRef}
            data-testid="tradingview-widget-container"
            style={{ minHeight: 520, width: "100%", borderRadius: 20, background: "#0f172a" }}
          />
        </div>
      </Space>
    </Card>
  );
}

function CalendarHistorySection({
  store,
}: {
  store: ReturnType<typeof useTradingPanelState>;
}) {
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
          return created.year() === displayedMonth.year() && created.month() === displayedMonth.month();
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
      <ul style={{ margin: 0, padding: 0, listStyle: "none" }}>
        {dayJournals.map((journal) => (
          <li key={journal.id} style={{ marginBottom: 4 }}>
            <AntBadge
              status={journalBadgeStatus(journal)}
              text={
                <Button
                  type="link"
                  size="small"
                  style={{
                    padding: 0,
                    height: "auto",
                    fontSize: 12,
                    fontWeight: selectedJournalId === journal.id ? 700 : 500,
                    color:
                      selectedJournalId === journal.id
                        ? "var(--app-accent)"
                        : "var(--app-text-soft)",
                  }}
                  onClick={(e) => {
                    e.stopPropagation();
                    setSelectedJournalId(journal.id);
                  }}
                >
                  {journal.symbol || "未命名"}
                </Button>
              }
            />
          </li>
        ))}
      </ul>
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
              <Text strong style={{ color: "var(--app-text)" }}>
                Journal 详情
              </Text>
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
            { key: "status", label: "状态", children: journal.status === "open" ? "进行中" : "已完成" },
          ]}
        />

        {journal.summary ? (
          <Paragraph style={{ margin: 0, color: "var(--app-muted)", lineHeight: 1.8 }}>
            {journal.summary}
          </Paragraph>
        ) : null}

        <Divider style={{ margin: 0, borderColor: "var(--app-border)" }} />

        <TimelineSection journal={journal} title="Journal 时间线" />
      </Space>
    </Card>
  );
}

function SettingsSection({
  store,
}: {
  store: ReturnType<typeof useTradingPanelState>;
}) {
  const { config } = store.state;
  const [presetSymbol, setPresetSymbol] = useState("");
  const [presetTimeframe, setPresetTimeframe] = useState<TimeframePreset>("5m");

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

          <Card style={panelCardStyle}>
            <Space orientation="vertical" size={12} style={{ width: "100%" }}>
              <Text strong style={{ color: "var(--app-text)" }}>交易对预设</Text>
              <Paragraph style={{ margin: 0, color: "var(--app-muted)" }}>
                为常用标的设置默认周期，并选择一个默认预设用于新建时自动带出。
              </Paragraph>

              <Input
                placeholder="例如 BTCUSDT"
                value={presetSymbol}
                onChange={(event) => setPresetSymbol(event.target.value)}
              />

              <Select
                value={presetTimeframe}
                options={TIMEFRAME_PRESET_VALUES.map((value) => ({ value, label: value }))}
                onChange={(value) => setPresetTimeframe(value)}
              />

              <Button
                type="primary"
                onClick={() => {
                  store.actions.upsertPairPreset(presetSymbol, presetTimeframe);
                  setPresetSymbol("");
                  setPresetTimeframe("5m");
                }}
              >
                保存预设
              </Button>

              <Space orientation="vertical" size={8} style={{ width: "100%" }}>
                <Text strong style={{ color: "var(--app-text)" }}>默认预设</Text>
                {store.state.pairPresets.length === 0 ? (
                  <Text style={{ color: "var(--app-muted)" }}>还没有保存任何交易对预设。</Text>
                ) : (
                  store.state.pairPresets.map((preset) => {
                    const isDefault = store.state.defaultPresetSymbol === preset.symbol;

                    return (
                      <Flex key={preset.symbol} justify="space-between" align="center" gap={12}>
                        <Space align="center" size={8}>
                          <Text strong style={{ color: "var(--app-text)" }}>{preset.symbol}</Text>
                          <Tag color={tagColorForVariant("neutral")}>{preset.timeframe}</Tag>
                          {isDefault ? <Tag color={tagColorForVariant("accent")}>默认预设</Tag> : null}
                        </Space>

                        <Space size={8}>
                          <Button
                            size="small"
                            onClick={() => {
                              store.actions.setDefaultPresetSymbol(preset.symbol);
                            }}
                          >
                            设为默认
                          </Button>
                          <Button
                            size="small"
                            danger
                            onClick={() => {
                              store.actions.removePairPreset(preset.symbol);
                            }}
                          >
                            删除
                          </Button>
                        </Space>
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

function PanelSurface({ store }: { store: ReturnType<typeof useTradingPanelState> }) {
  const { state, actions } = store;
  const journal = selectCurrentJournal(state.journals);
  const reminder = useReminderClock(state.config.timerEnabled, state.config.timerInterval);
  const defaultPreset = state.defaultPresetSymbol
    ? state.pairPresets.find((preset) => preset.symbol === state.defaultPresetSymbol)
    : undefined;
  const [showStartJournal, setShowStartJournal] = useState(false);
  const [showFinishJournal, setShowFinishJournal] = useState(false);
  const [isPinned, setIsPinned] = useState(state.panelPinned);

  useEffect(() => {
    setIsPinned(state.panelPinned);
  }, [state.panelPinned]);

  useEffect(() => {
    if (!isTauriRuntime()) {
      return;
    }

    const currentWindow = getCurrentWindow();
    void currentWindow.setAlwaysOnTop(isPinned).catch(() => undefined);
  }, [isPinned]);

  usePersistPanelBounds();

  return (
    <main style={{ minHeight: "100vh", padding: 12 }}>
      <Flex vertical gap={16} style={{ margin: "0 auto", width: "100%", maxWidth: 430 }}>
        <Card style={panelCardStyle}>
          <Flex align="center" justify="space-between" gap={12}>
            <Button
              type={isPinned ? "primary" : "default"}
              shape="circle"
              icon={isPinned ? <PushpinFilled /> : <PushpinOutlined />}
              title={isPinned ? "当前保持置顶" : "当前允许被覆盖"}
              onClick={() => {
                const nextPinned = !isPinned;
                setIsPinned(nextPinned);
                actions.setPanelPinned(nextPinned);
              }}
            />

            <Space orientation="vertical" size={2} style={{ flex: 1, minWidth: 0 }}>
              <Text strong style={{ fontSize: 11, letterSpacing: "0.18em", color: "var(--app-muted-strong)", textTransform: "uppercase" }}>
                Quick Capture Panel
              </Text>
              <Text style={{ color: "var(--app-muted)" }}>
                提醒 {formatReminderStatus(state.config.timerEnabled, state.config.timerInterval)}
              </Text>
            </Space>

            <Card
              size="small"
              style={{
                borderRadius: 16,
                borderColor: reminder.isFlashing ? "var(--app-border-strong)" : "var(--app-border)",
                background: reminder.isFlashing ? "var(--app-accent-soft)" : "white",
                boxShadow: reminder.isFlashing ? "var(--app-shadow-soft)" : "none",
              }}
            >
              <Statistic
                value={formatClock(reminder.now)}
                styles={{
                  content: {
                    color: reminder.isFlashing ? "var(--app-accent)" : "var(--app-text)",
                    fontSize: 15,
                    fontWeight: 700,
                    letterSpacing: "0.08em",
                    fontVariantNumeric: "tabular-nums",
                  },
                }}
              />
            </Card>
          </Flex>
        </Card>

        <Card style={panelCardStyle}>
          <Space orientation="vertical" size={16} style={{ width: "100%" }}>
            {journal ? (
              <>
                <Flex justify="space-between" align="start" gap={12}>
                  <Space orientation="vertical" size={8}>
                    <Flex align="center" gap={8} wrap="wrap">
                      <Title level={3} style={{ margin: 0, color: "var(--app-text)" }}>
                        {journal.symbol || "待填写标的"}
                      </Title>
                      <Tag color={tagColorForVariant(journal.status === "open" ? "accent" : "neutral")}>
                        {journal.status === "open" ? "进行中" : "已完成"}
                      </Tag>
                    </Flex>
                    <Text style={{ color: "var(--app-muted)" }}>
                      {journal.market || "未标记市场"} · 净仓 {formatNumber(netPosition(journal.events))}
                    </Text>
                  </Space>

                  {journal.status === "open" ? (
                    <Button size="small" onClick={() => setShowFinishJournal(true)}>
                      结束 Journal
                    </Button>
                  ) : null}
                </Flex>

                {journal.summary ? (
                  <Card style={mutedShellCardStyle}>
                    <Paragraph style={{ margin: 0, color: "var(--app-muted)", lineHeight: 1.8 }}>
                      {journal.summary}
                    </Paragraph>
                  </Card>
                ) : (
                  <Card style={dashedShellCardStyle}>
                    <Text style={{ color: "var(--app-muted)", lineHeight: 1.8 }}>
                      这条 journal 还没有总结，建议在收盘后补上执行结论与改进点。
                    </Text>
                  </Card>
                )}
              </>
            ) : (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={
                    <Space orientation="vertical" size={4}>
                    <Text strong style={{ color: "var(--app-text)" }}>
                      没有活跃 Journal
                    </Text>
                    <Text style={{ color: "var(--app-muted)" }}>
                      先创建今天的交易记录，再把每次观察、进出场与情绪变化逐条补进去。
                    </Text>
                  </Space>
                }
              >
                <Button type="primary" onClick={() => setShowStartJournal(true)}>
                  开始记录今天的交易
                </Button>
              </Empty>
            )}
          </Space>
        </Card>

        <TimelineSection journal={journal} title="最近记录" />

        <QuickCaptureCard
          journal={journal}
          onCreateJournal={() => setShowStartJournal(true)}
          onRecord={(draft) =>
            journal
              ? store.actions.recordEvent(journal.id, draft)
              : { ok: false, error: "missingContent" }
          }
        />
      </Flex>

      <StartJournalModal
        open={showStartJournal}
        defaultPreset={defaultPreset}
        onClose={() => setShowStartJournal(false)}
        onCreate={(symbol, market) => {
          actions.createJournal(symbol, market);
          setShowStartJournal(false);
        }}
      />

      <FinishJournalModal
        open={showFinishJournal && journal !== undefined}
        initialSummary={journal?.summary ?? ""}
        onClose={() => setShowFinishJournal(false)}
        onConfirm={(summary) => {
          if (!journal) {
            return;
          }

          actions.finishJournal(journal.id, summary);
          setShowFinishJournal(false);
        }}
      />
    </main>
  );
}

function TimelineSection({
  journal,
  title,
}: {
  journal: TradeJournal | undefined;
  title: string;
}) {
  const events = useMemo(
    () =>
      journal
        ? [...journal.events].sort(
            (left, right) =>
              new Date(right.timestamp).getTime() - new Date(left.timestamp).getTime(),
          )
        : [],
    [journal],
  );
  const highlightedId = useHighlightedTopEvent(events);

  return (
    <Card style={panelCardStyle}>
      <Flex justify="space-between" align="start" gap={12} style={{ marginBottom: 16 }}>
        <Space orientation="vertical" size={4}>
          <Title level={3} style={{ margin: 0, color: "var(--app-text)" }}>
            {title}
          </Title>
          <Paragraph style={{ margin: 0, color: "var(--app-muted)" }}>
            把同一笔交易拆成事件链，而不是只记结果。
          </Paragraph>
        </Space>
        <Tag color={tagColorForVariant("neutral")}>{events.length}</Tag>
      </Flex>

      <Space orientation="vertical" size={12} style={{ width: "100%" }}>
        {events.length === 0 ? (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description={
              <Space orientation="vertical" size={4}>
                <Text strong style={{ color: "var(--app-text)" }}>
                  准备记录下一条观察
                </Text>
                <Text style={{ color: "var(--app-muted)" }}>
                  你的最新进场、加减仓和笔记会按时间顺序出现在这里。
                </Text>
              </Space>
            }
          />
        ) : (
          events.map((event) => (
            <Card
              key={event.id}
              style={{
                ...panelCardStyle,
                width: "100%",
                borderColor:
                  highlightedId === event.id ? "var(--app-border-strong)" : "var(--app-border)",
                boxShadow: highlightedId === event.id ? "var(--app-shadow-soft)" : "none",
              }}
            >
              <Space orientation="vertical" size={12} style={{ width: "100%" }}>
                <Flex align="start" justify="space-between" gap={12}>
                  <Space orientation="vertical" size={8}>
                    <Flex align="center" gap={8} wrap="wrap">
                      <Tag color={tagColorForVariant(eventVariant(event.type))}>
                        {EVENT_SYMBOLS[event.type]} {EVENT_LABELS[event.type]}
                      </Tag>
                      <Text strong style={{ color: "var(--app-text)" }}>
                        {journal?.symbol || "Unknown"}
                      </Text>
                    </Flex>
                    <Flex wrap="wrap" gap={8}>
                      <Text style={{ color: "var(--app-muted)", fontSize: 12 }}>
                        {formatClock(new Date(event.timestamp))}
                      </Text>
                      {journal?.market ? (
                        <Text style={{ color: "var(--app-muted)", fontSize: 12 }}>
                          {journal.market}
                        </Text>
                      ) : null}
                      {event.quantity !== undefined ? (
                        <Tag color={tagColorForVariant("neutral")}>Qty {formatNumber(event.quantity)}</Tag>
                      ) : null}
                      {event.price !== undefined ? (
                        <Tag color={tagColorForVariant("neutral")}>@ {formatNumber(event.price)}</Tag>
                      ) : null}
                    </Flex>
                  </Space>
                </Flex>
                {event.content ? (
                  <Paragraph style={{ margin: 0, color: "var(--app-muted)", lineHeight: 1.8 }}>
                    {event.content}
                  </Paragraph>
                ) : null}
              </Space>
            </Card>
          ))
        )}
      </Space>
    </Card>
  );
}

function QuickCaptureCard({
  journal,
  onCreateJournal,
  onRecord,
}: {
  journal: TradeJournal | undefined;
  onCreateJournal: () => void;
  onRecord: (draft: TradeEventDraft) => ReturnType<typeof composeTradeEvent>;
}) {
  const currentNetPosition = journal ? netPosition(journal.events) : 0;
  const status = journal?.status ?? "open";
  const types = availableEventTypes(status, currentNetPosition);

  const [selectedType, setSelectedType] = useState<TradeEventType>(types[0] ?? "log");
  const [quantityInput, setQuantityInput] = useState("");
  const [priceInput, setPriceInput] = useState("");
  const [content, setContent] = useState("");
  const [didSubmit, setDidSubmit] = useState(false);

  useEffect(() => {
    if (!types.includes(selectedType)) {
      setSelectedType(types[0] ?? "log");
    }
  }, [selectedType, types]);

  const draft = useMemo<TradeEventDraft>(
    () => ({
      type: selectedType,
      quantityInput,
      priceInput,
      content,
    }),
    [content, priceInput, quantityInput, selectedType],
  );

  const validation = journal
    ? composeTradeEvent(draft, currentNetPosition, status)
    : ({ ok: false, error: "missingContent" } as const);

  const canSubmit = journal !== undefined && validation.ok;

  return (
    <Card style={panelCardStyle}>
      <Space orientation="vertical" size={16} style={{ width: "100%" }}>
        <Flex justify="space-between" align="start" gap={12}>
          <Space orientation="vertical" size={4}>
            <Title level={3} style={{ margin: 0, color: "var(--app-text)", letterSpacing: "-0.02em" }}>
              快速记录
            </Title>
            <Paragraph style={{ margin: 0, color: "var(--app-muted)", lineHeight: 1.6 }}>
              {journal
                ? journal.status === "done"
                  ? "这条 journal 已完成，现在只允许补充笔记。"
                  : "把关键事件先记下来，复杂分析留到主界面复盘。"
                : "开始一条 journal 后，这里会切换成盘中快速记录模式。"}
            </Paragraph>
          </Space>
          <Tag color={tagColorForVariant("neutral")}>⌘↩ 提交</Tag>
        </Flex>

        {journal ? (
          <>
            <Segmented
              block
              value={selectedType}
              options={types.map((type) => ({
                value: type,
                label: EVENT_LABELS[type],
              }))}
              onChange={(value) => setSelectedType(value as TradeEventType)}
            />

            {selectedType !== "log" ? (
              <div
                style={{
                  display: "grid",
                  gap: 12,
                  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
                }}
              >
                <Form layout="vertical">
                  <Form.Item label="数量" style={{ marginBottom: 0 }}>
                    {selectedType === "exit" ? (
                      <Card size="small" style={mutedShellCardStyle}>
                        <Text style={{ color: "var(--app-text)" }}>{formatNumber(-currentNetPosition)}</Text>
                      </Card>
                    ) : (
                      <Input
                        value={quantityInput}
                        placeholder="例如 1.25 或 -1.25"
                        onChange={(event) => setQuantityInput(event.currentTarget.value)}
                      />
                    )}
                  </Form.Item>
                </Form>

                <Form layout="vertical">
                  <Form.Item label="价格" style={{ marginBottom: 0 }}>
                    <Input
                      value={priceInput}
                      placeholder="可选"
                      onChange={(event) => setPriceInput(event.currentTarget.value)}
                    />
                  </Form.Item>
                </Form>
              </div>
            ) : null}

            {selectedType !== "log" ? (
              <Text style={{ color: "var(--app-muted)" }}>
                {quantityHint(selectedType, status, currentNetPosition)}
              </Text>
            ) : null}

            <Form layout="vertical">
              <Form.Item
                label={selectedType === "log" ? "观察记录" : "补充说明"}
                style={{ marginBottom: 0 }}
              >
                <Input.TextArea
                  rows={5}
                  value={content}
                  placeholder={
                    selectedType === "log"
                      ? "例如：回踩 VWAP 后承接变强，第二次确认才进场。"
                      : "可选补充"
                  }
                  onChange={(event) => setContent(event.currentTarget.value)}
                />
              </Form.Item>
            </Form>

            <Flex align="center" justify="space-between" gap={12}>
              <Text style={{ color: didSubmit ? "var(--app-success)" : "var(--app-muted)" }}>
                {didSubmit
                  ? "已记录"
                  : feedbackMessage(selectedType, journal, validation, canSubmit)}
              </Text>

              <Button
                type="primary"
                disabled={!canSubmit}
                onClick={() => {
                  const result = onRecord(draft);
                  if (!result.ok) {
                    return;
                  }

                  setDidSubmit(true);
                  setSelectedType("log");
                  setQuantityInput("");
                  setPriceInput("");
                  setContent("");

                  window.setTimeout(() => {
                    setDidSubmit(false);
                  }, 1200);
                }}
              >
                {didSubmit ? "已记录" : "记录事件"}
              </Button>
            </Flex>
          </>
        ) : (
          <Card style={dashedShellCardStyle}>
            <Space orientation="vertical" size={16} style={{ width: "100%" }}>
              <Text style={{ color: "var(--app-muted)", lineHeight: 1.8 }}>
                先创建今天的 journal，快速记录区才会解锁。
              </Text>
              <Button block type="primary" onClick={onCreateJournal}>
                开始 Journal
              </Button>
            </Space>
          </Card>
        )}
      </Space>
    </Card>
  );
}

function StartJournalModal({
  open,
  defaultPreset,
  onClose,
  onCreate,
}: {
  open: boolean;
  defaultPreset?: TradingPairPreset;
  onClose: () => void;
  onCreate: (symbol: string, market: string) => void;
}) {
  const [form] = Form.useForm<{ symbol: string; market: string }>();

  useEffect(() => {
    if (open) {
      form.resetFields();
      form.setFieldsValue({
        symbol: defaultPreset?.symbol ?? "",
        market: "",
      });
    }
  }, [defaultPreset, form, open]);

  return (
    <Modal
      title="开始新的 Journal"
      open={open}
      destroyOnHidden
      okText="创建"
      cancelText="取消"
      onCancel={onClose}
      onOk={() =>
        form.validateFields().then((values) => {
          onCreate(values.symbol, values.market);
        })
      }
    >
      <Form form={form} layout="vertical">
        {defaultPreset ? (
          <Paragraph style={{ marginBottom: 16, color: "var(--app-muted)" }}>
            默认周期 {defaultPreset.timeframe}
          </Paragraph>
        ) : null}
        <Form.Item
          name="symbol"
          label="标的"
          rules={[{ required: true, message: "请先填写标的" }]}
        >
          <Input placeholder="例如 NQ / ES / BTCUSDT" />
        </Form.Item>
        <Form.Item name="market" label="市场">
          <Input placeholder="例如 CME / Binance" />
        </Form.Item>
      </Form>
    </Modal>
  );
}

function FinishJournalModal({
  open,
  initialSummary,
  onClose,
  onConfirm,
}: {
  open: boolean;
  initialSummary: string;
  onClose: () => void;
  onConfirm: (summary: string) => void;
}) {
  const [form] = Form.useForm<{ summary: string }>();

  useEffect(() => {
    if (open) {
      form.setFieldsValue({ summary: initialSummary });
    }
  }, [form, initialSummary, open]);

  return (
    <Modal
      title="结束并写下总结"
      open={open}
      destroyOnHidden
      okText="确认结束"
      cancelText="取消"
      onCancel={onClose}
      onOk={() =>
        form.validateFields().then((values) => {
          onConfirm(values.summary ?? "");
        })
      }
    >
      <Form form={form} layout="vertical">
        <Form.Item name="summary" label="复盘摘要">
          <Input.TextArea rows={5} placeholder="可选：这笔交易哪里做得好，哪里需要改进？" />
        </Form.Item>
      </Form>
    </Modal>
  );
}

function tagColorForVariant(variant: AppTagVariant) {
  switch (variant) {
    case "accent":
      return "blue";
    case "success":
      return "green";
    case "warning":
      return "orange";
    case "danger":
      return "red";
    case "neutral":
      return undefined;
  }
}

function eventVariant(type: TradeEventType): AppTagVariant {
  switch (type) {
    case "entry":
      return "success";
    case "add":
    case "reduce":
      return "warning";
    case "exit":
      return "danger";
    case "log":
      return "accent";
  }
}

function useHighlightedTopEvent(events: TradeEvent[]) {
  const previousIds = useRef<string[]>([]);
  const [highlightedId, setHighlightedId] = useState<string>();

  useEffect(() => {
    const currentIds = events.map((event) => event.id);
    const currentTop = currentIds[0];

    if (
      currentTop &&
      currentTop !== previousIds.current[0] &&
      !previousIds.current.includes(currentTop)
    ) {
      setHighlightedId(currentTop);
      const timer = window.setTimeout(() => setHighlightedId(undefined), 1400);
      previousIds.current = currentIds;
      return () => {
        window.clearTimeout(timer);
      };
    }

    previousIds.current = currentIds;
    return undefined;
  }, [events]);

  return highlightedId;
}

function useReminderClock(enabled: boolean, interval: number) {
  const [now, setNow] = useState(() => new Date());
  const [isFlashing, setIsFlashing] = useState(false);

  useEffect(() => {
    const timer = window.setInterval(() => {
      setNow(new Date());
    }, 1000);

    return () => {
      window.clearInterval(timer);
    };
  }, []);

  useEffect(() => {
    if (!enabled) {
      setIsFlashing(false);
      return;
    }

    const intervalMs = interval * 1000;
    const delay = intervalMs - (Date.now() % intervalMs);
    let repeatingTimer = 0;

    const trigger = () => {
      setIsFlashing(true);
      window.setTimeout(() => setIsFlashing(false), 1000);
    };

    const initialTimer = window.setTimeout(() => {
      trigger();
      repeatingTimer = window.setInterval(trigger, intervalMs);
    }, delay);

    return () => {
      window.clearTimeout(initialTimer);
      if (repeatingTimer) {
        window.clearInterval(repeatingTimer);
      }
    };
  }, [enabled, interval]);

  return { now, isFlashing };
}

function usePersistPanelBounds() {
  useEffect(() => {
    if (!isTauriRuntime()) {
      return;
    }

    const currentWindow = getCurrentWindow();
    const persistBounds = async () => {
      const [position, size] = await Promise.all([
        currentWindow.outerPosition(),
        currentWindow.outerSize(),
      ]);

      window.localStorage.setItem(
        PANEL_BOUNDS_STORAGE_KEY,
        serializePanelBounds({
          x: position.x,
          y: position.y,
          width: size.width,
          height: size.height,
        }),
      );
    };

    void persistBounds().catch(() => undefined);
    const unsubscribers: Array<() => void> = [];

    void currentWindow
      .onMoved(() => {
        void persistBounds().catch(() => undefined);
      })
      .then((unlisten) => {
        unsubscribers.push(unlisten);
      });

    void currentWindow
      .onResized(() => {
        void persistBounds().catch(() => undefined);
      })
      .then((unlisten) => {
        unsubscribers.push(unlisten);
      });

    return () => {
      unsubscribers.forEach((unlisten) => unlisten());
    };
  }, []);
}

function feedbackMessage(
  selectedType: TradeEventType,
  journal: TradeJournal,
  validation: ReturnType<typeof composeTradeEvent>,
  canSubmit: boolean,
) {
  if (journal.status === "done") {
    return "已完成的 journal 现在只接受补充笔记。";
  }

  if (!canSubmit) {
    return validation.ok
      ? "补全必要字段后即可记录。"
      : validationMessage(selectedType, validation.error);
  }

  if (selectedType === "exit") {
    return "可以记录平仓。";
  }

  if (selectedType === "log") {
    return "可以记录观察。";
  }

  return "可以记录事件。";
}

function validationMessage(type: TradeEventType, error: TradeEventComposerError) {
  switch (error) {
    case "missingContent":
      return type === "log" ? "观察记录需要填写文本。" : "请先补全必要字段。";
    case "missingQuantity":
      return type === "entry" ? "开仓需要填写带方向的数量。" : "请先填写数量。";
    case "invalidPrecision":
      return "数量和价格最多保留两位小数。";
    case "zeroQuantity":
      return "数量不能为 0。";
    case "wrongDirection":
      return type === "add" ? "加仓方向必须与当前仓位一致。" : "减仓方向必须朝 0 回补。";
    case "exceedsPosition":
      return "减仓数量不能超过剩余仓位。";
    case "doneOnlyLogs":
      return "已完成的 journal 现在只允许补充笔记。";
  }
}

function quantityHint(
  type: TradeEventType,
  status: TradeJournal["status"],
  currentNetPosition: number,
) {
  if (status === "done") {
    return "已完成的 journal 现在只允许补充笔记。";
  }

  switch (type) {
    case "entry":
      return "正数表示做多，负数表示做空；最多两位小数。";
    case "add":
      return currentNetPosition < 0
        ? "当前是空头仓位，加仓数量也必须保持负数。"
        : "当前是多头仓位，加仓数量也必须保持正数。";
    case "reduce":
      return currentNetPosition < 0
        ? "空头减仓时，数量应填写正数。"
        : "多头减仓时，数量应填写负数。";
    case "exit":
      return "平仓会自动使用全部剩余仓位。";
    case "log":
      return "";
  }
}

function formatClock(date: Date) {
  return date.toLocaleTimeString("en-GB", {
    hour12: false,
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function formatReminderStatus(enabled: boolean, interval: number) {
  return enabled ? `开启 · ${formatReminderInterval(interval)}` : "关闭";
}

function formatReminderInterval(interval: number) {
  const match = REMINDER_INTERVALS.find((preset) => preset.value === interval);
  return match ? match.label : `${Math.round(interval / 60)}m`;
}

function formatJournalDate(value: string) {
  return new Date(value).toLocaleString("zh-CN", {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

function formatNumber(value: number) {
  return value.toLocaleString("en-US", {
    maximumFractionDigits: 2,
    minimumFractionDigits: isZero(value % 1) ? 0 : 0,
  });
}

function isTauriRuntime(): boolean {
  return typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;
}

const REMINDER_INTERVALS = [
  { label: "1m", value: 60 },
  { label: "5m", value: 300 },
  { label: "15m", value: 900 },
  { label: "30m", value: 1800 },
  { label: "1h", value: 3600 },
];

const EVENT_LABELS: Record<TradeEventType, string> = {
  entry: "开仓",
  add: "加仓",
  reduce: "减仓",
  exit: "平仓",
  log: "笔记",
};

const EVENT_SYMBOLS: Record<TradeEventType, string> = {
  entry: "↗",
  add: "+",
  reduce: "−",
  exit: "×",
  log: "☰",
};
