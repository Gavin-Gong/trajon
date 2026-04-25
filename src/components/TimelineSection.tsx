import { Empty, Flex, Space, Tag, Typography } from "antd";
import { useEffect, useMemo, useRef, useState } from "react";

import type { TradeEvent, TradeJournal } from "../domain/journal";
import { formatClock, formatNumber } from "./JournalFormatters";
import { EVENT_LABELS, EVENT_SYMBOLS, eventVariant, tagColorForVariant } from "./EventPresentation";
import { plainPanelSectionStyle } from "./PanelStyles";

const { Paragraph, Text, Title } = Typography;

const plainTimelineEventStyle = {
  borderBottom: "1px solid rgba(15, 23, 42, 0.08)",
  paddingBottom: 12,
};

const timelineMetricTagStyle = {
  fontFamily: "monospace",
};

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
  useHighlightedTopEvent(events);

  return (
    <div style={plainPanelSectionStyle}>
      <Flex justify="space-between" align="start" gap={12} style={{ marginBottom: 16 }}>
        <Space orientation="vertical" size={4}>
          <Title level={5} style={{ margin: 0, color: "rgba(0, 0, 0, 0.88)" }}>
            {title}
          </Title>
        </Space>
        <Tag color={tagColorForVariant("neutral")}>{events.length}</Tag>
      </Flex>

      <Space orientation="vertical" size={12} style={{ width: "100%" }}>
        {events.length === 0 ? (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description={
              <Space orientation="vertical" size={4}>
                <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>
                  暂无记录
                </Text>
              </Space>
            }
          />
        ) : (
          events.map((event) => (
            <div key={event.id} style={{ ...plainTimelineEventStyle, width: "100%" }}>
              <Space orientation="vertical" size={12} style={{ width: "100%" }}>
                <Flex align="start" justify="space-between" gap={12}>
                  <Space orientation="vertical" size={8}>
                    <Flex align="center" gap={8} wrap="wrap">
                      <Tag color={tagColorForVariant(eventVariant(event.type))}>
                        {EVENT_SYMBOLS[event.type]} {EVENT_LABELS[event.type]}
                      </Tag>
                      <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>
                        {journal?.symbol || "Unknown"}
                      </Text>
                    </Flex>
                    <Flex wrap="wrap" gap={8}>
                      <Text style={{ color: "rgba(0, 0, 0, 0.45)", fontSize: 12 }}>
                        {formatClock(new Date(event.timestamp))}
                      </Text>
                      {journal?.market ? (
                        <Text style={{ color: "rgba(0, 0, 0, 0.45)", fontSize: 12 }}>{journal.market}</Text>
                      ) : null}
                      {event.quantity !== undefined && event.price !== undefined ? (
                        <Tag color={tagColorForVariant("neutral")} style={timelineMetricTagStyle}>
                          Qty {formatNumber(event.quantity)} @ {formatNumber(event.price)}
                        </Tag>
                      ) : null}
                      {event.quantity !== undefined && event.price === undefined ? (
                        <Tag color={tagColorForVariant("neutral")} style={timelineMetricTagStyle}>
                          Qty {formatNumber(event.quantity)}
                        </Tag>
                      ) : null}
                      {event.price !== undefined && event.quantity === undefined ? (
                        <Tag color={tagColorForVariant("neutral")} style={timelineMetricTagStyle}>
                          @ {formatNumber(event.price)}
                        </Tag>
                      ) : null}
                    </Flex>
                  </Space>
                </Flex>
                {event.content ? (
                  <Paragraph style={{ margin: 0, color: "rgba(0, 0, 0, 0.45)", lineHeight: 1.8 }}>
                    {event.content}
                  </Paragraph>
                ) : null}
              </Space>
            </div>
          ))
        )}
      </Space>
    </div>
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
