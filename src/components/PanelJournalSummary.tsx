import { InfoCircleOutlined } from "@ant-design/icons";
import { Button, Flex, Space, Tag, Tooltip, Typography } from "antd";

import { netPosition, type TradeJournal } from "../domain/journal";
import { formatNumber } from "./JournalFormatters";
import { tagColorForVariant } from "./EventPresentation";
import { plainDashedBlockStyle, plainMutedBlockStyle, plainPanelSectionStyle } from "./PanelStyles";

const { Paragraph, Text, Title } = Typography;

const infoIconStyle = {
  color: "rgba(0, 0, 0, 0.45)",
  cursor: "help",
  fontSize: 14,
};

export function PanelJournalSummary({
  journal,
  onFinish,
}: {
  journal: TradeJournal | undefined;
  onFinish: () => void;
}) {
  return (
    <div style={plainPanelSectionStyle}>
      <Space orientation="vertical" size={16} style={{ width: "100%" }}>
        {journal ? (
          <>
            <Flex justify="space-between" align="start" gap={12}>
              <Space orientation="vertical" size={8}>
                <Flex align="center" gap={8} wrap="wrap">
                  <Title level={5} style={{ margin: 0, color: "rgba(0, 0, 0, 0.88)" }}>
                    {journal.symbol || "待填写标的"}
                  </Title>
                  <Tag color={tagColorForVariant(journal.status === "open" ? "accent" : "neutral")}>
                    {journal.status === "open" ? "进行中" : "已完成"}
                  </Tag>
                </Flex>
                <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>
                  {journal.market || "未标记市场"} · 净仓 {formatNumber(netPosition(journal.events))}
                </Text>
              </Space>

              {journal.status === "open" ? (
                <Button size="small" onClick={onFinish}>
                  结束 Journal
                </Button>
              ) : null}
            </Flex>

            {journal.summary ? (
              <div style={plainMutedBlockStyle}>
                <Paragraph style={{ margin: 0, color: "rgba(0, 0, 0, 0.45)", lineHeight: 1.8 }}>
                  {journal.summary}
                </Paragraph>
              </div>
            ) : (
              <div style={plainDashedBlockStyle}>
                <Flex align="center" gap={6}>
                  <Text style={{ color: "rgba(0, 0, 0, 0.45)", lineHeight: 1.8 }}>暂无总结</Text>
                  <Tooltip title="建议在收盘后补上执行结论与改进点。">
                    <InfoCircleOutlined aria-label="总结说明" style={infoIconStyle} />
                  </Tooltip>
                </Flex>
              </div>
            )}
          </>
        ) : null}
      </Space>
    </div>
  );
}
