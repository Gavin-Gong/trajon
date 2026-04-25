import { DeleteOutlined, EditOutlined } from "@ant-design/icons";
import { Button, Flex, Popconfirm, Segmented, Select, Space, Table, Tag, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useMemo, useState } from "react";

import type { TradeJournal } from "../domain/journal";
import { tagColorForVariant } from "./EventPresentation";
import { formatJournalDate } from "./JournalFormatters";
import { JournalDrawer } from "./JournalDrawer";
import type { TradingJournalStore } from "./JournalStoreTypes";

const { Text } = Typography;
type JournalStatusFilter = "all" | TradeJournal["status"];

export function JournalListSection({ store }: { store: TradingJournalStore }) {
  const [selectedJournalId, setSelectedJournalId] = useState<string | undefined>();
  const [statusFilter, setStatusFilter] = useState<JournalStatusFilter>("all");
  const [tagFilter, setTagFilter] = useState<string[]>([]);
  const journals = useMemo(
    () =>
      [...store.state.journals].sort(
        (left, right) => new Date(right.updatedAt).getTime() - new Date(left.updatedAt).getTime(),
      ),
    [store.state.journals],
  );
  const filteredJournals = useMemo(
    () => journals.filter((journal) => {
      if (statusFilter !== "all" && journal.status !== statusFilter) {
        return false;
      }

      if (tagFilter.length === 0) {
        return true;
      }

      const journalTagSet = new Set(journal.tags.map((tag) => tag.toLowerCase()));
      return tagFilter.some((tag) => journalTagSet.has(tag.toLowerCase()));
    }),
    [journals, statusFilter, tagFilter],
  );
  const tagOptions = useMemo(
    () =>
      Array.from(
        journals.reduce((options, journal) => {
          for (const tag of journal.tags) {
            options.set(tag.toLowerCase(), tag);
          }
          return options;
        }, new Map<string, string>()),
      ).map(([, tag]) => ({ label: tag, value: tag })),
    [journals],
  );
  const columns: ColumnsType<TradeJournal> = [
    {
      title: "交易对",
      dataIndex: "symbol",
      key: "symbol",
        render: (_, journal) => (
        <Space orientation="vertical" size={2}>
          <Text strong>{journal.symbol || "未命名"}</Text>
          <Text type="secondary">{journal.market || "未标记市场"}</Text>
        </Space>
      ),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 120,
      render: (status: TradeJournal["status"]) => (
        <Tag color={tagColorForVariant(status === "open" ? "accent" : "neutral")}>
          {status === "open" ? "进行中" : "已完成"}
        </Tag>
      ),
    },
    {
      title: "事件数",
      key: "events",
      width: 100,
      render: (_, journal) => journal.events.length,
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 180,
      render: (value: string) => formatJournalDate(value),
    },
    {
      title: "更新时间",
      dataIndex: "updatedAt",
      key: "updatedAt",
      width: 180,
      render: (value: string) => formatJournalDate(value),
    },
    {
      title: "总结",
      dataIndex: "summary",
      key: "summary",
      ellipsis: true,
      render: (summary?: string) => summary || "-",
    },
    {
      title: "标签",
      dataIndex: "tags",
      key: "tags",
      width: 220,
      render: (tags: string[]) =>
        tags.length > 0 ? (
          <Space wrap size={[6, 6]}>
            {tags.map((tag) => (
              <Tag key={tag}>{tag}</Tag>
            ))}
          </Space>
        ) : (
          "-"
        ),
    },
    {
      title: "操作",
      key: "actions",
      width: 180,
      fixed: "right",
      render: (_, journal) => (
        <Flex gap={8}>
          <Button
            type="text"
            icon={<EditOutlined />}
            aria-label={`编辑 ${journal.symbol || "未命名"} journal`}
            onClick={(event) => {
              event.stopPropagation();
              setSelectedJournalId(journal.id);
            }}
          />
          <Popconfirm
            title="删除这条 Journal？"
            okText="删除"
            cancelText="取消"
            okButtonProps={{ danger: true }}
            onConfirm={async () => {
              await store.actions.deleteJournal(journal.id);
              setSelectedJournalId((current) => (current === journal.id ? undefined : current));
              void message.success("已删除 Journal");
            }}
          >
            <Button
              type="text"
              icon={<DeleteOutlined />}
              aria-label={`删除 ${journal.symbol || "未命名"} journal`}
              onClick={(event) => {
                event.stopPropagation();
              }}
            />
          </Popconfirm>
        </Flex>
      ),
    },
  ];

  return (
    <>
      <Flex justify="space-between" align="center" gap={16} wrap="wrap" style={{ marginBottom: 16 }}>
        <Flex align="center" gap={12} wrap="wrap">
          <Segmented<JournalStatusFilter>
            value={statusFilter}
            onChange={(value) => setStatusFilter(value)}
            options={[
              { label: "全部", value: "all" },
              { label: "进行中", value: "open" },
              { label: "已完成", value: "done" },
            ]}
          />
          <Select
            mode="multiple"
            allowClear
            placeholder="按标签筛选"
            aria-label="按标签筛选"
            style={{ minWidth: 220 }}
            options={tagOptions}
            value={tagFilter}
            onChange={(value) => setTagFilter(value)}
          />
        </Flex>
        <Text type="secondary">共 {filteredJournals.length} 条</Text>
      </Flex>

      <Table<TradeJournal>
        rowKey="id"
        columns={columns}
        dataSource={filteredJournals}
        pagination={{ pageSize: 10, showSizeChanger: false }}
        scroll={{ x: 1180 }}
      />

      <JournalDrawer
        store={store}
        open={Boolean(selectedJournalId)}
        journalId={selectedJournalId}
        onClose={() => setSelectedJournalId(undefined)}
      />
    </>
  );
}
