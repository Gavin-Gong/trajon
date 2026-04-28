import { ClockCircleOutlined, DeleteOutlined, EditOutlined } from "@ant-design/icons";
import { Button, Drawer, Empty, Flex, Popconfirm, Space, Tag, Typography, message } from "antd";

import type { TradeJournal } from "../domain/journal";
import { openPanelWindow } from "../platform/panelWindow";
import { tagColorForVariant } from "./EventPresentation";
import { formatJournalDate } from "./JournalFormatters";
import type { TradingJournalStore } from "./JournalStoreTypes";

const { Text } = Typography;

export function OpenJournalDrawer({
  open,
  journals,
  store,
  onClose,
}: {
  open: boolean;
  journals: TradeJournal[];
  store: TradingJournalStore;
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

              <Flex align="center" gap={4}>
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
                <Popconfirm
                  title="删除这条 Journal？"
                  okText="删除"
                  cancelText="取消"
                  okButtonProps={{ danger: true }}
                  onConfirm={async () => {
                    await store.actions.deleteJournal(journal.id);
                    void message.success("已删除 Journal");
                  }}
                >
                  <Button
                    type="text"
                    size="small"
                    icon={<DeleteOutlined />}
                    aria-label={`删除进行中的 journal-${journal.id}`}
                  />
                </Popconfirm>
              </Flex>
            </Flex>
          ))}
        </Space>
      )}
    </Drawer>
  );
}
