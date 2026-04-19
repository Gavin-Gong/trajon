import { Column, Line } from "@ant-design/plots";
import { Card, Empty, Space, Typography } from "antd";
import { useMemo } from "react";

import { realizedPnlAnalytics } from "../domain/analysis";
import { formatChartDateLabel, formatSignedNumber } from "./JournalFormatters";
import { dashedShellCardStyle, shellCardStyle } from "./PanelStyles";
import type { TradingJournalStore } from "./JournalStoreTypes";

const { Text, Title } = Typography;

export function AnalysisSection({ store }: { store: TradingJournalStore }) {
  const analytics = useMemo(() => realizedPnlAnalytics(store.state.journals), [store.state.journals]);

  if (analytics.equityCurve.length === 0) {
    return (
      <Card style={dashedShellCardStyle}>
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={
            <Space orientation="vertical" size={4}>
              <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>还没有可分析的已实现盈亏</Text>
              <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>
                先完成至少一笔 journal，分析页会显示累计盈利曲线和每日盈亏柱状图。
              </Text>
            </Space>
          }
        />
      </Card>
    );
  }

  const lineConfig = {
    data: analytics.equityCurve.map((item) => ({
      timestamp: item.timestamp,
      value: item.value,
    })),
    autoFit: true,
    height: 280,
    xField: (datum: { timestamp: string; value: number }) => new Date(datum.timestamp),
    yField: "value",
    axis: {
      x: { title: false },
      y: { title: false },
    },
    tooltip: {
      title: (datum: { timestamp: string }) => formatChartDateLabel(datum.timestamp),
      items: [{ channel: "y", name: "累计已实现盈亏", valueFormatter: formatSignedNumber }],
    },
    style: {
      lineWidth: 2,
      stroke: "#2563eb",
    },
    point: {
      size: 3,
      style: {
        fill: "#2563eb",
      },
    },
  };

  const columnConfig = {
    data: analytics.dailyPnl.map((item) => ({ dayKey: item.dayKey, value: item.value })),
    autoFit: true,
    height: 280,
    xField: "dayKey",
    yField: "value",
    colorField: (datum: { dayKey: string; value: number }) => (datum.value >= 0 ? "盈利" : "亏损"),
    scale: {
      color: {
        type: "ordinal",
        domain: ["盈利", "亏损"],
        range: ["#16a34a", "#dc2626"],
      },
    },
    axis: {
      x: { title: false },
      y: { title: false },
    },
    tooltip: {
      title: (datum: { dayKey: string }) => datum.dayKey,
      items: [{ channel: "y", name: "当日已实现盈亏", valueFormatter: formatSignedNumber }],
    },
    style: {
      radiusTopLeft: 4,
      radiusTopRight: 4,
    },
  };

  return (
    <Space orientation="vertical" size={20} style={{ width: "100%" }}>
      <div
        style={{
          display: "grid",
          gap: 16,
          gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))",
        }}
      >
        <Card style={shellCardStyle}>
          <Space orientation="vertical" size={4}>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>累计已实现盈亏</Text>
            <Title level={3} style={{ margin: 0, color: analytics.realizedTotal >= 0 ? "#15803d" : "#b91c1c" }}>
              {formatSignedNumber(analytics.realizedTotal)}
            </Title>
          </Space>
        </Card>

        <Card style={shellCardStyle}>
          <Space orientation="vertical" size={4}>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>盈利 / 亏损天数</Text>
            <Title level={3} style={{ margin: 0, color: "rgba(0, 0, 0, 0.88)" }}>
              {analytics.profitDays} / {analytics.lossDays}
            </Title>
          </Space>
        </Card>

        <Card style={shellCardStyle}>
          <Space orientation="vertical" size={4}>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>已统计 Journal</Text>
            <Title level={3} style={{ margin: 0, color: "rgba(0, 0, 0, 0.88)" }}>
              {analytics.journalCount}
            </Title>
          </Space>
        </Card>
      </div>

      <Card style={shellCardStyle}>
        <Space orientation="vertical" size={12} style={{ width: "100%" }}>
          <div>
            <Title level={5} style={{ margin: 0, color: "rgba(0, 0, 0, 0.88)" }}>累计已实现盈亏</Title>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>按平仓事件累计，不包含未平仓浮盈浮亏。</Text>
          </div>
          <Line {...lineConfig} />
        </Space>
      </Card>

      <Card style={shellCardStyle}>
        <Space orientation="vertical" size={12} style={{ width: "100%" }}>
          <div>
            <Title level={5} style={{ margin: 0, color: "rgba(0, 0, 0, 0.88)" }}>每日已实现盈亏</Title>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>按 UTC 日期聚合，盈利为绿色，亏损为红色。</Text>
          </div>
          <Column {...columnConfig} />
        </Space>
      </Card>
    </Space>
  );
}
