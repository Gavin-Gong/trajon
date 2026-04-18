import { PushpinFilled, PushpinOutlined } from "@ant-design/icons";
import { getCurrentWindow } from "@tauri-apps/api/window";
import {
  Button,
  Card,
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

import { closePanelWindow } from "../../platform/panelLifecycle";
import {
  availableEventTypes,
  composeTradeEvent,
  currentJournal as selectCurrentJournal,
  netPosition,
  type TradeEventComposerResult,
  type TradeEventDraft,
  type TradeEventType,
  type TradeJournal,
} from "../../domain/tradingJournalDomain";
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
  type TradingJournalStore,
  usePersistPanelBounds,
  useReminderClock,
} from "../../components/tradingJournal/TradingJournalShared";

const { Paragraph, Text, Title } = Typography;

export function TradingJournalPanel({
  store,
  presetSymbol,
}: {
  store: TradingJournalStore;
  presetSymbol?: string;
}) {
  const { state, actions } = store;
  const journal = selectCurrentJournal(state.journals);
  const reminder = useReminderClock(state.config.timerEnabled, state.config.timerInterval);
  const incomingPreset = presetSymbol
    ? state.pairPresets.find((preset) => preset.symbol === presetSymbol)
    : undefined;
  const defaultPreset = incomingPreset ??
    (state.defaultPresetSymbol
      ? state.pairPresets.find((preset) => preset.symbol === state.defaultPresetSymbol)
      : undefined);
  const [showFinishJournal, setShowFinishJournal] = useState(false);
  const [isPinned, setIsPinned] = useState(state.panelPinned);

  useEffect(() => {
    if (journal || !defaultPreset?.symbol) {
      return;
    }

    actions.createJournal(defaultPreset.symbol, "");
  }, [actions, defaultPreset?.symbol, journal]);

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
            ) : null}
          </Space>
        </Card>

        <TimelineSection journal={journal} title="最近记录" />

        <QuickCaptureCard
          journal={journal}
          onRecord={(draft) =>
            journal
              ? store.actions.recordEvent(journal.id, draft)
              : Promise.resolve({ ok: false, error: "missingContent" } as const)
          }
        />
      </Flex>

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
          void closePanelWindow();
        }}
      />
    </main>
  );
}

function QuickCaptureCard({
  journal,
  onRecord,
}: {
  journal: TradeJournal | undefined;
  onRecord: (draft: TradeEventDraft) => Promise<TradeEventComposerResult>;
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
                   void onRecord(draft).then((result) => {
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
                   });
                 }}
               >
                {didSubmit ? "已记录" : "记录事件"}
              </Button>
            </Flex>
          </>
        ) : (
          <Card style={dashedShellCardStyle}>
            <Text style={{ color: "var(--app-muted)", lineHeight: 1.8 }}>
              正在准备新的 journal...
            </Text>
          </Card>
        )}
      </Space>
    </Card>
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
