import { getCurrentWindow } from "@tauri-apps/api/window";
import { Card, Empty, Flex, Space, Tag, Typography } from "antd";
import type { CSSProperties } from "react";
import { useEffect, useMemo, useRef, useState } from "react";

import {
  composeTradeEvent,
  isZero,
  type TradeEvent,
  type TradeEventComposerError,
  type TradeEventType,
  type TradeJournal,
} from "../../domain/tradingJournalDomain";
import { serializePanelBounds } from "../../platform/panelWindow";
import { PANEL_BOUNDS_STORAGE_KEY } from "../../model/storage";
import { useTradingJournalState } from "../../stores/useTradingJournalState";
import type { AppTagVariant } from "./TradingJournalTypes";

const { Paragraph, Text, Title } = Typography;

export const shellCardStyle: CSSProperties = {};

export const mutedShellCardStyle: CSSProperties = {
  ...shellCardStyle,
  background: "var(--app-panel-muted)",
  boxShadow: "none",
};

export const dashedShellCardStyle: CSSProperties = {
  ...mutedShellCardStyle,
  background: "transparent",
};

export const panelCardStyle: CSSProperties = {
  ...shellCardStyle,
};

export type TradingJournalStore = ReturnType<typeof useTradingJournalState>;

export function TimelineSection({
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

function useHighlightedTopEvent(events: TradeEvent[]) {
  const previousIds = useRef<string[]>([]);
  const [highlightedId, setHighlightedId] = useState<string>();

  useEffect(() => {
    const currentIds = events.map((event) => event.id);
    const currentTop = currentIds[0];

    if (currentTop && currentTop !== previousIds.current[0] && !previousIds.current.includes(currentTop)) {
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

export function tagColorForVariant(variant: AppTagVariant) {
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

  return "neutral";
}

export function useReminderClock(enabled: boolean, interval: number) {
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

export function usePersistPanelBounds() {
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

export function feedbackMessage(
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

export function quantityHint(
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

export function formatClock(date: Date) {
  return date.toLocaleTimeString("en-GB", {
    hour12: false,
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

export function formatReminderStatus(enabled: boolean, interval: number) {
  return enabled ? `开启 · ${formatReminderInterval(interval)}` : "关闭";
}

function formatReminderInterval(interval: number) {
  const match = REMINDER_INTERVALS.find((preset) => preset.value === interval);
  return match ? match.label : `${Math.round(interval / 60)}m`;
}

export function formatJournalDate(value: string) {
  return new Date(value).toLocaleString("zh-CN", {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

export function formatNumber(value: number) {
  return value.toLocaleString("en-US", {
    maximumFractionDigits: 2,
    minimumFractionDigits: isZero(value % 1) ? 0 : 0,
  });
}

function isTauriRuntime(): boolean {
  return typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;
}

export const REMINDER_INTERVALS = [
  { label: "1m", value: 60 },
  { label: "5m", value: 300 },
  { label: "15m", value: 900 },
  { label: "30m", value: 1800 },
  { label: "1h", value: 3600 },
];

export const EVENT_LABELS: Record<TradeEventType, string> = {
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
