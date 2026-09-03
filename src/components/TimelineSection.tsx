import { ArrowDownAZ, ArrowUpAZ } from "lucide-react";
import { Button, Empty, Flex, Space, Tag, Tooltip, Typography } from "antd";
import { useEffect, useMemo, useRef, useState } from "react";

import type { TradeEvent, TradeJournal } from "../domain/journal";
import { monoFontFamily } from "../theme/useAppTheme";
import { formatClock, formatNumber } from "./JournalFormatters";
import { EVENT_ICONS, EVENT_LABELS, eventVariant, tagColorForVariant } from "./EventPresentation";
import { plainPanelSectionStyle } from "./PanelStyles";

const { Paragraph, Text, Title } = Typography;

const plainTimelineEventStyle = {
  borderBottom: "1px solid rgba(15, 23, 42, 0.08)",
  paddingBottom: 12,
};

const timelineMetricTagStyle = {
  fontFamily: monoFontFamily,
};

const eventTagContentStyle = {
  display: "inline-flex",
  alignItems: "center",
  gap: 4,
  verticalAlign: "middle",
};

const eventTagIconWrapStyle = {
  display: "inline-flex",
  alignItems: "center",
  justifyContent: "center",
  lineHeight: 0,
};

const eventContentQuoteStyle = {
  margin: 0,
  paddingLeft: 12,
  borderLeft: "3px solid rgba(15, 23, 42, 0.14)",
  color: "rgba(0, 0, 0, 0.88)",
  lineHeight: 1.8,
};

export function TimelineSection({
  journal,
  title,
}: {
  journal: TradeJournal | undefined;
  title: string;
}) {
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("asc");
  const events = useMemo(
    () =>
      journal
        ? [...journal.events].sort(
            (left, right) =>
              sortDirection === "asc"
                ? new Date(left.timestamp).getTime() - new Date(right.timestamp).getTime()
                : new Date(right.timestamp).getTime() - new Date(left.timestamp).getTime(),
          )
        : [],
    [journal, sortDirection],
  );
  useHighlightedTopEvent(events);

  const SortIcon = sortDirection === "asc" ? ArrowUpAZ : ArrowDownAZ;

  return (
    <div style={plainPanelSectionStyle}>
      <Flex justify="space-between" align="start" gap={12} style={{ marginBottom: 16 }}>
        <Space orientation="vertical" size={4}>
          <Title level={5} style={{ margin: 0, color: "rgba(0, 0, 0, 0.88)" }}>
            {title}
          </Title>
        </Space>
        <Tooltip title={sortDirection === "asc" ? "当前按时间正序，点击切换倒序" : "当前按时间倒序，点击切换正序"}>
          <Button
            type="text"
            aria-label="切换最近记录排序"
            icon={<SortIcon size={16} strokeWidth={2} />}
            onClick={() => setSortDirection((current) => (current === "asc" ? "desc" : "asc"))}
          />
        </Tooltip>
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
          events.map((event) => {
            const EventIcon = EVENT_ICONS[event.type];

            return (
              <div key={event.id} style={{ ...plainTimelineEventStyle, width: "100%" }}>
                <Space orientation="vertical" size={12} style={{ width: "100%" }}>
                  <Flex align="start" justify="space-between" gap={12}>
                    <Space orientation="vertical" size={8} style={{ width: "100%" }}>
                      <Flex align="center" gap={8} style={{ width: "100%" }}>
                        <Flex align="center" gap={8} wrap="wrap" style={{ flex: 1, minWidth: 0 }}>
                        <Tag color={tagColorForVariant(eventVariant(event.type))} style={{ fontFamily: monoFontFamily }}>
                          <span style={eventTagContentStyle}>
                            <span style={eventTagIconWrapStyle}>
                              <EventIcon aria-hidden size={14} strokeWidth={2} />
                            </span>
                            <span>{EVENT_LABELS[event.type]}</span>
                          </span>
                        </Tag>
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
                        <Text
                          style={{
                            color: "rgba(0, 0, 0, 0.45)",
                            fontSize: 12,
                            marginInlineStart: "auto",
                            whiteSpace: "nowrap",
                            fontFamily: monoFontFamily,
                          }}
                        >
                          {formatClock(new Date(event.timestamp))}
                        </Text>
                      </Flex>
                    </Space>
                  </Flex>
                  {event.content ? <Paragraph style={eventContentQuoteStyle}>{event.content}</Paragraph> : null}
                </Space>
              </div>
            );
          })
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
