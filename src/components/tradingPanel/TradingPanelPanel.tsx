import { PushpinFilled, PushpinOutlined } from "@ant-design/icons";
import { getCurrentWindow } from "@tauri-apps/api/window";
import {
  Button,
  Card,
  Empty,
  Flex,
  Form,
  Input,
  Modal,
  Segmented,
  Space,
  Statistic,
  Tag,
  Typography,
} from "antd";
import { useEffect, useMemo, useState } from "react";

import {
  availableEventTypes,
  composeTradeEvent,
  currentJournal as selectCurrentJournal,
  netPosition,
  type TradeEventDraft,
  type TradeEventType,
  type TradeJournal,
} from "../../tradingPanelDomain";
import {
  EVENT_LABELS,
  dashedShellCardStyle,
  feedbackMessage,
  formatClock,
  formatNumber,
  formatReminderStatus,
  mutedShellCardStyle,
  panelCardStyle,
  quantityHint,
  tagColorForVariant,
  TimelineSection,
  type TradingPanelStore,
  usePersistPanelBounds,
  useReminderClock,
} from "./TradingPanelShared";

const { Paragraph, Text, Title } = Typography;

export function PanelSurface({ store }: { store: TradingPanelStore }) {
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
    if (!(typeof window !== "undefined" && "__TAURI_INTERNALS__" in window)) {
      return;
    }

    const currentWindow = getCurrentWindow();
    void currentWindow.setAlwaysOnTop(isPinned).catch(() => undefined);
  }, [isPinned]);

  usePersistPanelBounds();

  return (
    <main style={{ minHeight: "100vh" }}>
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
              <Text
                strong
                style={{
                  fontSize: 11,
                  letterSpacing: "0.18em",
                  color: "var(--app-muted-strong)",
                  textTransform: "uppercase",
                }}
              >
                Quick Capture Panel
              </Text>
              <Text style={{ color: "var(--app-muted)" }}>
                提醒 {formatReminderStatus(state.config.timerEnabled, state.config.timerInterval)}
              </Text>
            </Space>

            <Card
              size="small"
              style={{
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
            journal ? store.actions.recordEvent(journal.id, draft) : { ok: false, error: "missingContent" }
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
                {didSubmit ? "已记录" : feedbackMessage(selectedType, journal, validation, canSubmit)}
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
  defaultPreset?: { symbol: string; timeframe: string };
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
