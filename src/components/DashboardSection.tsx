import { EditOutlined } from "@ant-design/icons";
import { Button, Card, Flex, Space, Tag, Typography } from "antd";

import { openPanelWindow } from "../platform/panelWindow";
import { tagColorForVariant } from "./EventPresentation";
import { formatJournalDate } from "./JournalFormatters";
import { shellCardStyle } from "./PanelStyles";
import type { TradingJournalStore } from "./JournalStoreTypes";

const { Text } = Typography;

export function DashboardSection({ store }: { store: TradingJournalStore }) {
  const recentJournals = [...store.state.journals]
    .sort((left, right) => new Date(right.updatedAt).getTime() - new Date(left.updatedAt).getTime())
    .slice(0, 3);

  return (
    <div
      data-testid="dashboard-panel-layout"
      style={{
        display: "grid",
        gap: 20,
        gridTemplateColumns: "minmax(0, 1fr) 280px",
        alignItems: "start",
      }}
    >
      <Space orientation="vertical" size={20} style={{ width: "100%" }}>
        <Card data-testid="dashboard-placeholder-card-1" style={shellCardStyle}>
          <div style={{ minHeight: 180 }} />
        </Card>

        <Card data-testid="dashboard-placeholder-card-2" style={shellCardStyle}>
          <div style={{ minHeight: 220 }} />
        </Card>
      </Space>

      <Card data-testid="dashboard-journal-list-card" style={shellCardStyle}>
        <Space orientation="vertical" size={16} style={{ width: "100%" }}>
          {recentJournals.map((journal, index) => (
            <div
              key={journal.id}
              style={{
                paddingBottom: index === recentJournals.length - 1 ? 0 : 12,
                borderBottom:
                  index === recentJournals.length - 1 ? "none" : "1px solid rgba(15, 23, 42, 0.08)",
              }}
            >
              <Flex justify="space-between" align="start" gap={12}>
                <Space orientation="vertical" size={6} style={{ minWidth: 0 }}>
                  <Flex align="center" gap={8} wrap="wrap">
                    <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>{journal.symbol || "未命名"}</Text>
                    <Tag color={tagColorForVariant(journal.status === "open" ? "accent" : "neutral")}>
                      {journal.status === "open" ? "进行中" : "已完成"}
                    </Tag>
                  </Flex>
                  <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>更新于 {formatJournalDate(journal.updatedAt)}</Text>
                </Space>

                <Button
                  type="text"
                  size="small"
                  icon={<EditOutlined />}
                  aria-label={`编辑-journal-${journal.id}`}
                  onClick={() => {
                    void openPanelWindow({ journalId: journal.id });
                  }}
                />
              </Flex>
            </div>
          ))}
        </Space>
      </Card>
    </div>
  );
}
