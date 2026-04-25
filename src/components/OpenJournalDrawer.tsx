import { ClockCircleOutlined, EditOutlined } from "@ant-design/icons";
import { Button, Drawer, Empty, Flex, Space, Tag, Typography } from "antd";

import type { TradeJournal } from "../domain/journal";
import { openPanelWindow } from "../platform/panelWindow";
import { tagColorForVariant } from "./EventPresentation";
import { formatJournalDate } from "./JournalFormatters";

const { Text } = Typography;

export function OpenJournalDrawer({
  open,
  journals,
  onClose,
}: {
  open: boolean;
  journals: TradeJournal[];
  onClose: () => void;
}) {
  return (
    <Drawer
      title="进行中的 Journal"
      placement="right"
      size="default"
      closable={{ placement: "end" }}
      open={open}
      onClose={onClose}
    >
      {journals.length === 0 ? (
        <Empty description="暂无进行中的 Journal" image={Empty.PRESENTED_IMAGE_SIMPLE} />
      ) : (
        <Space orientation="vertical" size={12} style={{ width: "100%" }}>
          {journals.map((journal, index) => (
            <Flex
              key={journal.id}
              justify="space-between"
              align="start"
              gap={12}
              style={{
                paddingBottom: index === journals.length - 1 ? 0 : 12,
                borderBottom: index === journals.length - 1 ? "none" : "1px solid rgba(15, 23, 42, 0.08)",
              }}
            >
              <Space orientation="vertical" size={6} style={{ minWidth: 0, flex: 1 }}>
                <Flex align="center" gap={8} wrap="wrap">
                  <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>
                    {journal.symbol || "未命名"}
                  </Text>
                  <Tag color={tagColorForVariant("accent")}>进行中</Tag>
                </Flex>
                <Flex align="center" gap={6}>
                  <ClockCircleOutlined style={{ color: "rgba(0, 0, 0, 0.45)" }} />
                  <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>更新于 {formatJournalDate(journal.updatedAt)}</Text>
                </Flex>
              </Space>

              <Button
                type="text"
                size="small"
                icon={<EditOutlined />}
                aria-label={`继续记录-journal-${journal.id}`}
                onClick={() => {
                  void openPanelWindow({ journalId: journal.id });
                  onClose();
                }}
              />
            </Flex>
          ))}
        </Space>
      )}
    </Drawer>
  );
}
