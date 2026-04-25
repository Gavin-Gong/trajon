import { Column, Line } from "@ant-design/plots";
import { Card, Empty, Space, Tooltip, Typography } from "antd";
import { useMemo } from "react";

import { realizedPnlAnalytics } from "../domain/analysis";
import { formatChartDateLabel, formatSignedNumber } from "./JournalFormatters";
import { dashedShellCardStyle, shellCardStyle } from "./PanelStyles";
import type { TradingJournalStore } from "./JournalStoreTypes";

const { Text, Title } = Typography;
const HEATMAP_MONTH_LABELS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
const HEATMAP_DAY_LABELS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

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
                先完成至少一笔 journal，分析页会显示累计盈利曲线、每日盈亏柱状图和年度热力图。
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

  const heatmapWeeks = Array.from({ length: Math.ceil(analytics.heatmap.length / 7) }, (_, weekIndex) =>
    analytics.heatmap.slice(weekIndex * 7, weekIndex * 7 + 7),
  );
  const monthMarkers = heatmapWeeks
    .map((week, weekIndex) => {
      const firstDay = week[0];
      if (!firstDay) {
        return null;
      }

      const date = new Date(`${firstDay.dayKey}T00:00:00.000Z`);
      return date.getUTCDate() <= 7 ? { weekIndex, label: HEATMAP_MONTH_LABELS[date.getUTCMonth()] } : null;
    })
    .filter((marker): marker is { weekIndex: number; label: string } => marker !== null);

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

      <Card style={shellCardStyle}>
        <Space orientation="vertical" size={12} style={{ width: "100%" }}>
          <div>
            <Title level={5} style={{ margin: 0, color: "rgba(0, 0, 0, 0.88)" }}>近一年盈亏热力图</Title>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>按 UTC 日期展示近 53 周已实现盈亏，绿色越深代表盈利越高，红色越深代表亏损越大。</Text>
          </div>

          <div style={{ overflowX: "auto", paddingBottom: 4 }}>
            <div style={{ display: "inline-flex", gap: 8, minWidth: "fit-content" }}>
              <div style={{ width: 28, paddingTop: 18 }}>
                {HEATMAP_DAY_LABELS.map((label, dayIndex) => (
                  <div
                    key={label}
                    style={{
                      height: 14,
                      marginBottom: dayIndex === HEATMAP_DAY_LABELS.length - 1 ? 0 : 4,
                      fontSize: 11,
                      lineHeight: "14px",
                      color: dayIndex % 2 === 1 ? "rgba(0, 0, 0, 0.45)" : "transparent",
                    }}
                  >
                    {label}
                  </div>
                ))}
              </div>

              <div>
                <div style={{ position: "relative", height: 18, marginBottom: 8 }}>
                  {monthMarkers.map((marker) => (
                    <div
                      key={`${marker.label}-${marker.weekIndex}`}
                      style={{
                        position: "absolute",
                        left: marker.weekIndex * 16,
                        top: 0,
                        fontSize: 11,
                        color: "rgba(0, 0, 0, 0.45)",
                      }}
                    >
                      {marker.label}
                    </div>
                  ))}
                </div>

                <div style={{ display: "flex", gap: 4 }}>
                  {heatmapWeeks.map((week, weekIndex) => (
                    <div key={`week-${weekIndex}`} style={{ display: "grid", gap: 4 }}>
                      {week.map((cell) => (
                        <Tooltip
                          key={cell.dayKey}
                          title={(
                            <Space orientation="vertical" size={0}>
                              <Text style={{ color: "inherit" }}>{cell.dayKey}</Text>
                              <Text style={{ color: "inherit" }}>当日已实现盈亏 {formatSignedNumber(cell.value)}</Text>
                            </Space>
                          )}
                        >
                          <div
                            role="gridcell"
                            aria-label={`${cell.dayKey} ${formatSignedNumber(cell.value)}`}
                            style={{
                              width: 12,
                              height: 12,
                              borderRadius: 3,
                              background: heatmapColor(cell.direction, cell.bucket),
                              boxShadow: cell.bucket === 0 ? "inset 0 0 0 1px rgba(15, 23, 42, 0.06)" : "none",
                            }}
                          />
                        </Tooltip>
                      ))}
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>

          <Space size={16} wrap>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>Less</Text>
            <Space size={4}>
              {[0, 1, 2, 3, 4].map((bucket) => (
                <div key={`profit-${bucket}`} style={{ width: 12, height: 12, borderRadius: 3, background: heatmapColor("profit", bucket) }} />
              ))}
            </Space>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>Profit</Text>
            <Space size={4}>
              {[0, 1, 2, 3, 4].map((bucket) => (
                <div key={`loss-${bucket}`} style={{ width: 12, height: 12, borderRadius: 3, background: heatmapColor("loss", bucket) }} />
              ))}
            </Space>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>Loss</Text>
          </Space>
        </Space>
      </Card>
    </Space>
  );
}

function heatmapColor(direction: "profit" | "loss" | "flat", bucket: number): string {
  if (direction === "flat" || bucket === 0) {
    return "#e5e7eb";
  }

  const profitPalette = ["#dcfce7", "#86efac", "#4ade80", "#22c55e"];
  const lossPalette = ["#fee2e2", "#fca5a5", "#f87171", "#ef4444"];
  const palette = direction === "profit" ? profitPalette : lossPalette;
  return palette[Math.max(0, Math.min(bucket - 1, palette.length - 1))] ?? "#e5e7eb";
}
