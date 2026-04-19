import { Button, Card, Descriptions, Divider, Empty, Flex, Space, Tag, Typography } from "antd";

import { netPosition, type TradeJournal } from "../domain/journal";
import { openPanelWindow } from "../platform/panelWindow";
import { TimelineSection } from "./TimelineSection";
import { tagColorForVariant } from "./EventPresentation";
import { formatJournalDate, formatNumber } from "./JournalFormatters";
import { dashedShellCardStyle, shellCardStyle } from "./PanelStyles";

const { Paragraph, Text, Title } = Typography;

export function JournalDetailPanel({ journal }: { journal: TradeJournal | undefined }) {
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
          <Paragraph style={{ margin: 0, color: "rgba(0, 0, 0, 0.45)", lineHeight: 1.8 }}>{journal.summary}</Paragraph>
        ) : null}

        <Button
          type="default"
          onClick={() => {
            void openPanelWindow({ journalId: journal.id });
          }}
        >
          继续编辑这条 Journal
        </Button>

        <Divider style={{ margin: 0 }} />

        <TimelineSection journal={journal} title="Journal 时间线" />
      </Space>
    </Card>
  );
}
