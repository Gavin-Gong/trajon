import { InfoCircleOutlined } from "@ant-design/icons";
import { Button, Flex, Form, Input, InputNumber, Segmented, Space, Tooltip, Typography } from "antd";
import { useEffect, useMemo, useState } from "react";

import {
  availableEventTypes,
  composeTradeEvent,
  netPosition,
  type TradeEventComposerResult,
  type TradeEventDraft,
  type TradeEventType,
  type TradeJournal,
} from "../domain/journal";
import { EVENT_LABELS } from "./EventPresentation";
import { feedbackMessage, quantityHint } from "./EventFeedback";
import { formatNumber } from "./JournalFormatters";
import { plainDashedBlockStyle, plainMutedBlockStyle, plainPanelSectionStyle } from "./PanelStyles";

const { Text, Title } = Typography;

const infoIconStyle = {
  color: "rgba(0, 0, 0, 0.45)",
  cursor: "help",
  fontSize: 14,
};

export function QuickCaptureCard({
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
  const [quantityInput, setQuantityInput] = useState<number | null>(null);
  const [priceInput, setPriceInput] = useState<number | null>(null);
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
        quantityInput: quantityInput == null ? "" : String(quantityInput),
        priceInput: priceInput == null ? "" : String(priceInput),
        content,
      }),
    [content, priceInput, quantityInput, selectedType],
  );

  const validation = journal
    ? composeTradeEvent(draft, currentNetPosition, status)
    : ({ ok: false, error: "missingContent" } as const);

  const canSubmit = journal !== undefined && validation.ok;

  return (
    <div style={plainPanelSectionStyle}>
      <Space orientation="vertical" size={16} style={{ width: "100%" }}>
        <Flex justify="space-between" align="start" gap={12}>
          <Space orientation="vertical" size={4}>
            <Flex align="center" gap={6} wrap="wrap">
              <Title level={5} style={{ margin: 0, color: "rgba(0, 0, 0, 0.88)", letterSpacing: "-0.02em" }}>
                快速记录
              </Title>
              <Tooltip
                title={journal
                  ? journal.status === "done"
                    ? "这条 journal 已完成，现在只允许补充笔记。"
                    : "把关键事件先记下来，复杂面板内容留到主界面复盘。"
                  : "开始一条 journal 后，这里会切换成盘中快速记录模式。"}
              >
                <InfoCircleOutlined aria-label="快速记录说明" style={infoIconStyle} />
              </Tooltip>
            </Flex>
          </Space>
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

            {selectedType !== "log" && selectedType !== "plan" ? (
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
                      <div style={plainMutedBlockStyle}>
                        <Text style={{ color: "rgba(0, 0, 0, 0.88)" }}>{formatNumber(-currentNetPosition)}</Text>
                      </div>
                    ) : (
                      <InputNumber
                        value={quantityInput}
                        placeholder="例如 1.25 或 -1.25"
                        controls
                        changeOnWheel
                        precision={2}
                        style={{ width: "100%" }}
                        onChange={(value) => setQuantityInput(typeof value === "number" ? value : null)}
                      />
                    )}
                  </Form.Item>
                </Form>

                <Form layout="vertical">
                  <Form.Item label="价格" style={{ marginBottom: 0 }}>
                    <InputNumber
                      value={priceInput}
                      placeholder="可选"
                      controls
                      changeOnWheel
                      precision={2}
                      min={0}
                      parser={(value) => value?.replace(/-/g, "") ?? ""}
                      style={{ width: "100%" }}
                      onChange={(value) => setPriceInput(typeof value === "number" ? value : null)}
                    />
                  </Form.Item>
                </Form>
              </div>
            ) : null}

            {selectedType !== "log" && selectedType !== "plan" ? (
              <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>{quantityHint(selectedType, status, currentNetPosition)}</Text>
            ) : null}

            <Form layout="vertical">
              <Form.Item
                label={selectedType === "log" ? "观察记录" : selectedType === "plan" ? "计划记录" : "补充说明"}
                style={{ marginBottom: 0 }}
              >
                <Input.TextArea
                  rows={5}
                  value={content}
                  placeholder={
                    selectedType === "log"
                      ? "例如：回踩 VWAP 后承接变强，第二次确认才进场。"
                      : selectedType === "plan"
                        ? "例如：等待回踩确认后再开仓，若失守关键位则放弃。"
                      : "可选补充"
                  }
                  onChange={(event) => setContent(event.currentTarget.value)}
                />
              </Form.Item>
            </Form>

            <Flex align="center" justify="space-between" gap={12}>
              <Text style={{ color: didSubmit ? "#52c41a" : "rgba(0, 0, 0, 0.45)" }}>
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
                    setQuantityInput(null);
                    setPriceInput(null);
                    setContent("");

                    window.setTimeout(() => {
                      setDidSubmit(false);
                    }, 1200);
                  });
                }}
              >
                {didSubmit ? "已记录" : "记录"}
              </Button>
            </Flex>
          </>
        ) : (
          <div style={plainDashedBlockStyle}>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)", lineHeight: 1.8 }}>正在准备新的 journal...</Text>
          </div>
        )}
      </Space>
    </div>
  );
}
