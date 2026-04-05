import { getCurrentWindow } from "@tauri-apps/api/window";
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
  Col,
  FloatButton,
  Form,
  Input,
  Layout,
  Menu,
  Modal,
  Row,
  Select,
  Switch,
  Tag,
} from "antd";
import type { BadgeProps as AntBadgeProps, CalendarProps } from "antd";
import dayjs, { type Dayjs } from "dayjs";
import "dayjs/locale/zh-cn";
import { useEffect, useMemo, useRef, useState } from "react";

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

dayjs.locale("zh-cn");
import {
  openPanelWindow,
  PANEL_BOUNDS_STORAGE_KEY,
  serializePanelBounds,
} from "./panelWindow";
import { useTradingPanelState } from "./tradingPanelStore";

type AppView = "main" | "panel";
type AppSection = "analytics" | "calendar" | "settings";
type AppTagVariant = "accent" | "danger" | "neutral" | "success" | "warning";

const fieldClassName =
  "w-full rounded-2xl border border-[var(--app-border)] bg-white px-4 py-3 text-sm text-[var(--app-text)] shadow-sm outline-none transition focus:border-[var(--app-border-strong)] focus:ring-4 focus:ring-[var(--app-ring)] placeholder:text-slate-400";

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
    <main className="min-h-screen">
      <Layout>
        <Row gutter={[24, 24]} wrap className="min-h-screen">
          <Col xs={24} lg={7} xl={6}>
            <Layout.Sider
              width="100%"
              breakpoint={undefined}
              collapsedWidth={0}
              theme="light"
              className="!h-full"
            >
              <div className="flex h-full flex-col gap-6">
                <div className="space-y-4">
                  <div className="inline-flex items-center rounded-full bg-[var(--app-accent-soft)] px-3 py-1 text-xs font-semibold tracking-[0.18em] text-[var(--app-accent)] uppercase">
                    Trading Flow
                  </div>
                  <h1 className="text-[clamp(1.9rem,2vw,2.6rem)] font-semibold tracking-[-0.04em] text-[var(--app-text)]">
                    Trading Flow
                  </h1>
                </div>

                <Menu
                  mode="inline"
                  selectedKeys={[section]}
                  items={menuItems}
                  onClick={({ key }) => setSection(key as AppSection)}
                  className="border-0 bg-transparent"
                />
              </div>
            </Layout.Sider>
          </Col>

          <Col xs={24} lg={17} xl={18}>
            <section className="min-w-0 py-1">
              {section === "analytics" ? <AnalyticsPanel store={store} /> : null}
              {section === "calendar" ? <CalendarHistorySection store={store} /> : null}
              {section === "settings" ? <SettingsSection store={store} /> : null}
            </section>
          </Col>
        </Row>
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
  const journals = store.state.journals;
  const current = selectCurrentJournal(journals);
  const openJournals = journals.filter((journal) => journal.status === "open");
  const monthJournals = journals.filter((journal) =>
    isSameMonth(new Date(journal.createdAt), new Date()),
  );
  const totalEvents = journals.reduce((count, journal) => count + journal.events.length, 0);
  const doneCount = journals.filter((journal) => journal.status === "done").length;
  const recentJournals = [...journals].sort(compareJournalForRecent).slice(0, 4);

  return (
    <div className="grid gap-6">
      <div className="grid gap-8 p-7 lg:grid-cols-[minmax(0,1.35fr)_minmax(320px,0.9fr)] lg:p-8">
        <div className="space-y-5">
          <div className="inline-flex items-center rounded-full border border-[var(--app-border)] bg-white px-3 py-1 text-xs font-semibold tracking-[0.18em] text-[var(--app-muted-strong)] uppercase">
            Intraday Review Workspace
          </div>
          <div className="space-y-4">
            <h2 className="text-[clamp(2rem,3vw,3.25rem)] font-semibold tracking-[-0.05em] text-[var(--app-text)]">
              盘中复盘工作台
            </h2>
            <p className="max-w-[56ch] text-sm leading-8 text-[var(--app-muted)] sm:text-[15px]">
              先把执行过程留住，再把情绪、仓位变化与结论整理出来。这个主页专门服务于日内交易员的记录与复盘节奏。
            </p>
          </div>
          <div className="flex flex-wrap gap-3">
            <Button type="primary" onClick={() => void openPanelWindow()}>
              打开记录面板
            </Button>
            <Button>查看本月沉淀</Button>
          </div>
        </div>

        <div className="grid gap-4">
          <div className="rounded-[26px] border border-white/70 bg-white/85 p-5 shadow-[var(--app-shadow-soft)]">
            <p className="text-xs font-semibold tracking-[0.16em] text-[var(--app-muted-strong)] uppercase">
              Today Snapshot
            </p>
            <div className="mt-4 grid grid-cols-2 gap-3">
              <MetricTile label="活跃 Journal" value={openJournals.length} tone="accent" />
              <MetricTile label="已完成" value={doneCount} tone="success" />
              <MetricTile label="事件记录" value={totalEvents} tone="neutral" />
              <MetricTile label="本月 Journal" value={monthJournals.length} tone="warning" />
            </div>
          </div>

          <div className="rounded-[26px] border border-[var(--app-border)] bg-[var(--app-surface-strong)] p-5 shadow-[var(--app-shadow-soft)]">
            <div className="flex items-center justify-between gap-3">
              <div>
                <p className="text-xs font-semibold tracking-[0.16em] text-[var(--app-muted-strong)] uppercase">
                  当前追踪标的
                </p>
                <p className="mt-2 text-2xl font-semibold tracking-[-0.04em] text-[var(--app-text)]">
                  {current?.symbol ? `当前 ${current.symbol}` : "暂无"}
                </p>
              </div>
              <Tag color={tagColorForVariant(current ? "accent" : "neutral")}>
                {current ? (current.status === "open" ? "进行中" : "待复盘") : "空闲"}
              </Tag>
            </div>
            <p className="mt-4 text-sm leading-7 text-[var(--app-muted)]">
              {current
                ? `${current.market || "未标记市场"} · 当前净仓 ${formatNumber(netPosition(current.events))}`
                : "没有活跃 journal 时，建议先打开记录面板开始当天第一条记录。"}
            </p>
          </div>
        </div>
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1.15fr)_minmax(320px,0.85fr)]">
        <div>
          <div className="mb-4">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div className="space-y-2">
                <h3 className="text-lg font-semibold tracking-[-0.02em] text-[var(--app-text)]">今日执行概览</h3>
                <p className="text-sm text-[var(--app-muted)]">
                  先看活跃 journal、当前仓位和关键记录，再决定今天复盘的重心。
                </p>
              </div>
              <div className="flex flex-wrap gap-2">
                <Tag color={tagColorForVariant("accent")}>未完成 {openJournals.length}</Tag>
                <Tag color={tagColorForVariant("neutral")}>总 Journal {journals.length}</Tag>
              </div>
            </div>
          </div>
          <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_280px]">
            <div className="space-y-4 rounded-[24px] border border-[var(--app-border)] bg-[var(--app-panel-muted)] p-5">
              <p className="text-sm font-medium text-[var(--app-muted-strong)]">活跃标的列表</p>
              {openJournals.length > 0 ? (
                <div className="flex flex-wrap gap-2">
                  {openJournals.map((journal) => (
                    <span
                      key={journal.id}
                      className="inline-flex items-center rounded-full border border-[var(--app-border)] bg-white px-3 py-1.5 text-sm font-semibold text-[var(--app-text)]"
                    >
                      {journal.symbol || "未命名"}
                    </span>
                  ))}
                </div>
              ) : (
                <p className="text-sm leading-7 text-[var(--app-muted)]">
                  当前没有活跃标的，适合回到历史 journal 做深一点的复盘归因。
                </p>
              )}
            </div>

            <div className="rounded-[24px] border border-[var(--app-border)] bg-white p-5">
              <p className="text-sm font-medium text-[var(--app-muted-strong)]">当前建议</p>
              <ul className="mt-3 grid gap-3 text-sm leading-7 text-[var(--app-muted)]">
                <li>• 先记录进出场依据，再补情绪与执行偏差。</li>
                <li>• 盘中只追求简洁，不在记录面板里做复杂分析。</li>
                <li>• 复盘时优先看事件链，而不是只看最终盈亏。</li>
              </ul>
            </div>
          </div>
        </div>

        <div>
          <div className="mb-4">
            <h3 className="text-lg font-semibold tracking-[-0.02em] text-[var(--app-text)]">本月日志沉淀</h3>
            <p className="mt-1 text-sm text-[var(--app-muted)]">
              把当月 journal 按最近时间归拢，快速判断哪些样本值得继续深挖。
            </p>
          </div>
          <div className="space-y-3">
            {recentJournals.length > 0 ? (
              recentJournals.map((journal) => (
                <div
                  key={journal.id}
                  className="flex items-center justify-between gap-3 rounded-[22px] border border-[var(--app-border)] bg-[var(--app-panel-muted)] px-4 py-3"
                >
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-semibold text-[var(--app-text)]">
                        {journal.symbol ? `${journal.symbol} Journal` : "未命名"}
                      </span>
                      <Tag color={tagColorForVariant(journal.status === "open" ? "accent" : "neutral")}>
                        {journal.status === "open" ? "进行中" : "已完成"}
                      </Tag>
                    </div>
                    <p className="mt-1 truncate text-sm text-[var(--app-muted)]">
                      {journal.market || "未标记市场"} · {formatJournalDate(journal.createdAt)}
                    </p>
                  </div>
                  <span className="text-sm font-medium text-[var(--app-muted-strong)]">
                    {journal.events.length} 条
                  </span>
                </div>
              ))
            ) : (
              <div className="rounded-[24px] border border-dashed border-[var(--app-border)] bg-[var(--app-panel-muted)] p-5 text-sm leading-7 text-[var(--app-muted)]">
                本月还没有可复盘的记录，先从一笔完整 journal 开始。
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
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
      <ul className="m-0 list-none p-0">
        {dayJournals.map((journal) => (
          <li key={journal.id} className="mb-1">
            <AntBadge
              status={journalBadgeStatus(journal)}
              text={
                <button
                  type="button"
                  className={[
                    "cursor-pointer border-none bg-transparent p-0 text-xs transition",
                    selectedJournalId === journal.id
                      ? "font-bold text-[var(--app-accent)]"
                      : "font-medium text-[var(--app-text-soft)] hover:text-[var(--app-accent)]",
                  ].join(" ")}
                  onClick={(e) => {
                    e.stopPropagation();
                    setSelectedJournalId(journal.id);
                  }}
                >
                  {journal.symbol || "未命名"}
                </button>
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
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_380px]">
      <div>
        <div className="mb-4 flex flex-wrap items-end justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold tracking-[-0.02em] text-[var(--app-text)]">日历回看</h2>
            <p className="mt-1 text-sm text-[var(--app-muted)]">
              按创建日期浏览历史 journal，点击查看细节链路。
            </p>
          </div>
          <Tag color={tagColorForVariant("neutral")}>{monthJournals.length} 条</Tag>
        </div>
        <Calendar
          mode="month"
          value={displayedMonth}
          cellRender={cellRender}
          onPanelChange={(date) => setDisplayedMonth(date)}
          onSelect={(date) => setDisplayedMonth(date)}
        />
      </div>

      <JournalDetailPanel journal={selectedJournal} />
    </div>
  );
}

function JournalDetailPanel({ journal }: { journal: TradeJournal | undefined }) {
  if (!journal) {
    return (
      <div className="h-fit rounded-xl border border-dashed border-[var(--app-border)] px-5 py-8 text-center">
        <p className="text-sm font-semibold text-[var(--app-text)]">Journal 详情</p>
        <p className="mt-2 text-sm text-[var(--app-muted)]">从左侧日历选择一条 journal，这里会展示时间线与总结。</p>
      </div>
    );
  }

  return (
    <div className="grid h-fit gap-5">
      <div>
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <div className="flex items-center gap-2">
              <h3 className="text-base font-semibold text-[var(--app-text)]">{journal.symbol || "未命名"}</h3>
              <Tag color={tagColorForVariant(journal.status === "open" ? "accent" : "neutral")}>
                {journal.status === "open" ? "进行中" : "已完成"}
              </Tag>
            </div>
            <p className="mt-1 text-sm text-[var(--app-muted)]">
              {journal.market || "未标记市场"} · {formatJournalDate(journal.createdAt)}
            </p>
          </div>
          <span className="rounded-full bg-[var(--app-panel-muted)] px-3 py-1 text-xs font-semibold text-[var(--app-muted-strong)]">
            净仓 {formatNumber(netPosition(journal.events))}
          </span>
        </div>
        {journal.summary ? (
          <p className="mt-3 text-sm leading-7 text-[var(--app-muted)]">{journal.summary}</p>
        ) : null}
      </div>

      <div className="h-px bg-[var(--app-border)]" />

      <TimelineSection journal={journal} title="Journal 时间线" />
    </div>
  );
}

function SettingsSection({
  store,
}: {
  store: ReturnType<typeof useTradingPanelState>;
}) {
  const { config } = store.state;

  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_320px]">
      <div>
        <div className="mb-4">
          <h3 className="text-lg font-semibold tracking-[-0.02em] text-[var(--app-text)]">设置</h3>
          <p className="mt-1 text-sm text-[var(--app-muted)]">
            提醒频率保存在本地，用来支持浮动面板的盘中记录节奏。
          </p>
        </div>
        <div className="grid gap-4">
          <div className="flex items-center justify-between gap-4 rounded-[24px] border border-[var(--app-border)] bg-[var(--app-panel-muted)] px-5 py-4">
            <div>
              <p className="text-sm font-semibold text-[var(--app-text)]">启用提醒</p>
              <p className="mt-1 text-sm text-[var(--app-muted)]">按固定频率提醒自己记录盘中观察。</p>
            </div>
            <Switch
              checked={config.timerEnabled}
              onChange={(checked) => {
                store.actions.updateConfig({ timerEnabled: checked });
              }}
            />
          </div>

          <div className="rounded-[24px] border border-[var(--app-border)] bg-white p-5">
            <p className="text-sm font-semibold text-[var(--app-text)]">提醒间隔</p>
            <p className="mt-1 text-sm text-[var(--app-muted)]">建议与自己的盘中节奏保持一致，避免提醒过密造成打断。</p>
            <div className="mt-4 max-w-[280px]">
              <Select
                className="w-full"
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
            </div>
          </div>
        </div>
      </div>

      <div className="h-fit">
        <div className="mb-4">
          <h3 className="text-lg font-semibold tracking-[-0.02em] text-[var(--app-text)]">数据与导出</h3>
          <p className="mt-1 text-sm text-[var(--app-muted)]">目前数据保存在本地。导出入口保留在这里，便于后续接入 CSV。</p>
        </div>
        <div className="space-y-4">
          <div className="rounded-[24px] border border-dashed border-[var(--app-border)] bg-[var(--app-panel-muted)] p-4 text-sm leading-7 text-[var(--app-muted)]">
            当前版本优先把记录链路和复盘体验打磨稳定，导出能力后续接入。
          </div>
          <Button className="w-full" onClick={() => undefined}>
            导出 CSV（即将支持）
          </Button>
        </div>
      </div>
    </div>
  );
}

function PanelSurface({ store }: { store: ReturnType<typeof useTradingPanelState> }) {
  const { state, actions } = store;
  const journal = selectCurrentJournal(state.journals);
  const reminder = useReminderClock(state.config.timerEnabled, state.config.timerInterval);
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
    <main className="min-h-screen px-3 py-3">
      <div className="mx-auto grid min-h-[calc(100vh-1.5rem)] w-full max-w-[430px] gap-4">
        <div className="flex items-center justify-between gap-3 p-4">
            <button
              type="button"
              className={[
                "inline-flex size-11 items-center justify-center rounded-2xl border text-sm font-semibold transition",
                isPinned
                  ? "border-[var(--app-border-strong)] bg-[var(--app-accent-soft)] text-[var(--app-accent)]"
                  : "border-[var(--app-border)] bg-white text-[var(--app-muted-strong)]",
              ].join(" ")}
              title={isPinned ? "当前保持置顶" : "当前允许被覆盖"}
              onClick={() => {
                const nextPinned = !isPinned;
                setIsPinned(nextPinned);
                actions.setPanelPinned(nextPinned);
              }}
            >
              {isPinned ? "📌" : "⊙"}
            </button>

            <div className="min-w-0 flex-1">
              <p className="text-[11px] font-semibold tracking-[0.18em] text-[var(--app-muted-strong)] uppercase">
                Quick Capture Panel
              </p>
              <p className="mt-1 text-sm text-[var(--app-muted)]">
                提醒 {formatReminderStatus(state.config.timerEnabled, state.config.timerInterval)}
              </p>
            </div>

            <div
              className={[
                "rounded-2xl border px-4 py-3 text-sm font-semibold tabular-nums tracking-[0.08em]",
                reminder.isFlashing
                  ? "border-[var(--app-border-strong)] bg-[var(--app-accent-soft)] text-[var(--app-accent)] shadow-[var(--app-shadow-soft)]"
                  : "border-[var(--app-border)] bg-white text-[var(--app-text)]",
              ].join(" ")}
            >
              {formatClock(reminder.now)}
            </div>
        </div>

        <div>
          <div className="mb-3">
            {journal ? (
              <div className="flex items-start justify-between gap-3">
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <h3 className="text-base font-semibold text-[var(--app-text)]">{journal.symbol || "待填写标的"}</h3>
                    <Tag color={tagColorForVariant(journal.status === "open" ? "accent" : "neutral")}>
                      {journal.status === "open" ? "进行中" : "已完成"}
                    </Tag>
                  </div>
                  <p className="text-sm text-[var(--app-muted)]">
                    {journal.market || "未标记市场"} · 净仓 {formatNumber(netPosition(journal.events))}
                  </p>
                </div>

                {journal.status === "open" ? (
                  <Button size="small" onClick={() => setShowFinishJournal(true)}>
                    结束 Journal
                  </Button>
                ) : null}
              </div>
            ) : (
              <div className="space-y-3">
                <h3 className="text-base font-semibold text-[var(--app-text)]">没有活跃 Journal</h3>
                <p className="text-sm text-[var(--app-muted)]">
                  先创建今天的交易记录，再把每次观察、进出场与情绪变化逐条补进去。
                </p>
              </div>
            )}
          </div>
          <div>
            {journal ? (
              journal.summary ? (
                <div className="rounded-[22px] bg-[var(--app-panel-muted)] px-4 py-3 text-sm leading-7 text-[var(--app-muted)]">
                  {journal.summary}
                </div>
              ) : (
                <div className="rounded-[22px] border border-dashed border-[var(--app-border)] px-4 py-3 text-sm leading-7 text-[var(--app-muted)]">
                  这条 journal 还没有总结，建议在收盘后补上执行结论与改进点。
                </div>
              )
            ) : (
                <Button className="w-full" type="primary" onClick={() => setShowStartJournal(true)}>
                  开始记录今天的交易
                </Button>
            )}
          </div>
        </div>

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
      </div>

      <StartJournalModal
        open={showStartJournal}
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
    <div>
      <div className="mb-4 flex items-center justify-between gap-3">
        <div>
          <h3 className="text-base font-semibold text-[var(--app-text)]">{title}</h3>
          <p className="mt-1 text-sm text-[var(--app-muted)]">把同一笔交易拆成事件链，而不是只记结果。</p>
        </div>
        <Tag color={tagColorForVariant("neutral")}>{events.length}</Tag>
      </div>
      <div className="grid gap-3">
        {events.length === 0 ? (
          <div className="rounded-[24px] border border-dashed border-[var(--app-border)] bg-[var(--app-panel-muted)] p-6 text-center">
            <p className="text-sm font-semibold text-[var(--app-text)]">准备记录下一条观察</p>
            <p className="mt-2 text-sm leading-7 text-[var(--app-muted)]">
              你的最新进场、加减仓和笔记会按时间顺序出现在这里。
            </p>
          </div>
        ) : (
          events.map((event) => (
            <article
              key={event.id}
              className={[
                "rounded-[24px] border bg-white p-4 transition",
                highlightedId === event.id
                  ? "border-[var(--app-border-strong)] shadow-[var(--app-shadow-soft)]"
                  : "border-[var(--app-border)]",
              ].join(" ")}
            >
              <div className="flex items-start justify-between gap-3">
                <div className="space-y-2">
                  <div className="flex flex-wrap items-center gap-2">
                    <Tag color={tagColorForVariant(eventVariant(event.type))}>
                      {EVENT_SYMBOLS[event.type]} {EVENT_LABELS[event.type]}
                    </Tag>
                    <span className="text-sm font-semibold text-[var(--app-text)]">
                      {journal?.symbol || "Unknown"}
                    </span>
                  </div>
                  <div className="flex flex-wrap gap-2 text-xs text-[var(--app-muted)]">
                    <span>{formatClock(new Date(event.timestamp))}</span>
                    {journal?.market ? <span>{journal.market}</span> : null}
                    {event.quantity !== undefined ? (
                      <span className="rounded-full bg-[var(--app-panel-muted)] px-2 py-1 text-[var(--app-muted-strong)]">
                        Qty {formatNumber(event.quantity)}
                      </span>
                    ) : null}
                    {event.price !== undefined ? (
                      <span className="rounded-full bg-[var(--app-panel-muted)] px-2 py-1 text-[var(--app-muted-strong)]">
                        @ {formatNumber(event.price)}
                      </span>
                    ) : null}
                  </div>
                </div>
              </div>
              {event.content ? (
                <p className="mt-3 text-sm leading-7 text-[var(--app-muted)]">{event.content}</p>
              ) : null}
            </article>
          ))
        )}
      </div>
    </div>
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
    <div className="rounded-[28px] border border-[var(--app-border)] bg-white shadow-[var(--app-shadow-soft)]">
      <div className="flex flex-col gap-2 p-6">
        <div className="flex items-start justify-between gap-3">
          <div>
            <h3 className="text-lg font-semibold tracking-[-0.02em] text-[var(--app-text)]">快速记录</h3>
            <p className="text-sm leading-6 text-[var(--app-muted)]">
              {journal
                ? journal.status === "done"
                  ? "这条 journal 已完成，现在只允许补充笔记。"
                  : "把关键事件先记下来，复杂分析留到主界面复盘。"
                : "开始一条 journal 后，这里会切换成盘中快速记录模式。"}
            </p>
          </div>
          <span className="rounded-full bg-[var(--app-panel-muted)] px-3 py-1 text-xs font-semibold text-[var(--app-muted-strong)]">
            ⌘↩ 提交
          </span>
        </div>
      </div>
      <div className="grid gap-4 px-6 pb-6">
        {journal ? (
          <>
            <div className="flex flex-wrap gap-2">
              {types.map((type) => (
                <button
                  type="button"
                  key={type}
                  className={[
                    "rounded-full border px-3 py-2 text-xs font-semibold transition",
                    type === selectedType
                      ? "border-[var(--app-border-strong)] bg-[var(--app-accent-soft)] text-[var(--app-accent)]"
                      : "border-[var(--app-border)] bg-white text-[var(--app-muted-strong)] hover:bg-[var(--app-panel-muted)]",
                  ].join(" ")}
                  onClick={() => setSelectedType(type)}
                >
                  {EVENT_LABELS[type]}
                </button>
              ))}
            </div>

            {selectedType !== "log" ? (
              <div className="grid gap-3 sm:grid-cols-2">
                <label className="grid gap-2 text-sm font-medium text-[var(--app-text-soft)]">
                  <span>数量</span>
                  {selectedType === "exit" ? (
                    <div className="rounded-2xl border border-[var(--app-border)] bg-[var(--app-panel-muted)] px-4 py-3 text-sm text-[var(--app-text)]">
                      {formatNumber(-currentNetPosition)}
                    </div>
                  ) : (
                    <input
                      className={fieldClassName}
                      value={quantityInput}
                      placeholder="例如 1.25 或 -1.25"
                      onChange={(event) => setQuantityInput(event.currentTarget.value)}
                    />
                  )}
                </label>

                <label className="grid gap-2 text-sm font-medium text-[var(--app-text-soft)]">
                  <span>价格</span>
                  <input
                    className={fieldClassName}
                    value={priceInput}
                    placeholder="可选"
                    onChange={(event) => setPriceInput(event.currentTarget.value)}
                  />
                </label>
              </div>
            ) : null}

            {selectedType !== "log" ? (
              <p className="text-sm text-[var(--app-muted)]">
                {quantityHint(selectedType, status, currentNetPosition)}
              </p>
            ) : null}

            <label className="grid gap-2 text-sm font-medium text-[var(--app-text-soft)]">
              <span>{selectedType === "log" ? "观察记录" : "补充说明"}</span>
              <textarea
                className={`${fieldClassName} min-h-[108px] resize-y`}
                value={content}
                placeholder={
                  selectedType === "log"
                    ? "例如：回踩 VWAP 后承接变强，第二次确认才进场。"
                    : "可选补充"
                }
                onChange={(event) => setContent(event.currentTarget.value)}
              />
            </label>

            <div className="flex items-center justify-between gap-3">
              <p className={["text-sm", didSubmit ? "text-[var(--app-success)]" : "text-[var(--app-muted)]"].join(" ")}>
                {didSubmit
                  ? "已记录"
                  : feedbackMessage(selectedType, journal, validation, canSubmit)}
              </p>

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
            </div>
          </>
        ) : (
          <div className="grid gap-4 rounded-[24px] border border-dashed border-[var(--app-border)] bg-[var(--app-panel-muted)] p-5">
            <p className="text-sm leading-7 text-[var(--app-muted)]">先创建今天的 journal，快速记录区才会解锁。</p>
            <Button className="w-full" type="primary" onClick={onCreateJournal}>
              开始 Journal
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}

function StartJournalModal({
  open,
  onClose,
  onCreate,
}: {
  open: boolean;
  onClose: () => void;
  onCreate: (symbol: string, market: string) => void;
}) {
  const [form] = Form.useForm<{ symbol: string; market: string }>();

  useEffect(() => {
    if (open) {
      form.resetFields();
    }
  }, [form, open]);

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

function MetricTile({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
  tone: "accent" | "success" | "warning" | "neutral";
}) {
  const toneClasses: Record<typeof tone, string> = {
    accent: "bg-[var(--app-accent-soft)] text-[var(--app-accent)]",
    success: "bg-[var(--app-success-soft)] text-[var(--app-success)]",
    warning: "bg-[var(--app-warning-soft)] text-[var(--app-warning)]",
    neutral: "bg-slate-100 text-slate-700",
  };

  return (
    <div className={["rounded-[22px] px-4 py-4", toneClasses[tone]].join(" ")}>
      <p className="text-xs font-semibold tracking-[0.16em] uppercase opacity-80">{label}</p>
      <p className="mt-2 text-2xl font-semibold tracking-[-0.04em]">{value}</p>
    </div>
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

function compareJournalForRecent(left: TradeJournal, right: TradeJournal) {
  return new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime();
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

function isSameMonth(left: Date, right: Date) {
  return (
    left.getUTCFullYear() === right.getUTCFullYear() &&
    left.getUTCMonth() === right.getUTCMonth()
  );
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
