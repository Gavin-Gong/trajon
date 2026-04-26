import { InfoCircleOutlined } from "@ant-design/icons";
import { Column, Line } from "@ant-design/plots";
import { Card, Empty, Space, Tooltip, Typography } from "antd";
import { useMemo } from "react";

import { realizedPnlAnalytics, strategyAnalytics } from "../domain/analysis";
import { formatChartDateLabel, formatSignedNumber } from "./JournalFormatters";
import { dashedShellCardStyle, shellCardStyle } from "./PanelStyles";
import type { TradingJournalStore } from "./JournalStoreTypes";

const { Text, Title } = Typography;
const HEATMAP_MONTH_LABELS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
const HEATMAP_DAY_LABELS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

const topAnalysisGridStyle = {
  display: "grid",
  gap: 16,
  gridTemplateColumns: "minmax(0, 1fr) minmax(0, 2fr)",
  alignItems: "stretch",
};

const topMetricGridStyle = {
  display: "grid",
  gap: 12,
  gridTemplateColumns: "repeat(5, minmax(0, 1fr))",
};

const smallCardGridStyle = {
  display: "grid",
  gap: 12,
  gridTemplateColumns: "repeat(auto-fit, minmax(150px, 1fr))",
  height: "100%",
};

const compactMetricCardStyle = {
  ...shellCardStyle,
};

const compactMetricStackStyle = {
  width: "100%",
};

const cardTitleStyle = {
  margin: 0,
  color: "rgba(0, 0, 0, 0.88)",
};

const cardTitleRowStyle = {
  alignItems: "center",
  display: "inline-flex",
  gap: 6,
};

const cardDescriptionIconStyle = {
  color: "rgba(0, 0, 0, 0.45)",
  cursor: "help",
  fontSize: 14,
};

export function AnalysisSection({ store }: { store: TradingJournalStore }) {
  const analytics = useMemo(() => realizedPnlAnalytics(store.state.journals), [store.state.journals]);
  const strategy = useMemo(() => strategyAnalytics(store.state.journals), [store.state.journals]);

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
    data: analytics.dailyPnl.reduce<Array<{ dayKey: string; value: number }>>((result, item) => {
      const previousValue = result[result.length - 1]?.value ?? 0;
      result.push({ dayKey: item.dayKey, value: previousValue + item.value });
      return result;
    }, []),
    autoFit: true,
    height: 208,
    xField: "dayKey",
    yField: "value",
    axis: {
      x: { title: false },
      y: { title: false },
    },
    tooltip: {
      title: (datum: { dayKey: string }) => formatChartDateLabel(`${datum.dayKey}T00:00:00.000Z`),
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
    scale: {
      y: {
        nice: true,
      },
      color: {
        type: "ordinal",
        domain: ["盈利", "亏损"],
        range: ["#16a34a", "#dc2626"],
      },
    },
    colorField: (datum: { dayKey: string; value: number }) => (datum.value >= 0 ? "盈利" : "亏损"),
    axis: {
      x: { title: false },
      y: { title: false },
    },
    tooltip: {
      title: (datum: { dayKey: string }) => datum.dayKey,
      items: [{ channel: "y", name: "当日已实现盈亏", valueFormatter: formatSignedNumber }],
    },
    style: {
      maxWidth: 20,
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
      <div style={topMetricGridStyle}>
        <Card style={compactMetricCardStyle} styles={{ body: { padding: 14 } }}>
          <Space orientation="vertical" size={2} style={compactMetricStackStyle}>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>总盈亏</Text>
            <Title level={4} style={{ margin: 0, color: analytics.realizedTotal >= 0 ? "#15803d" : "#b91c1c" }}>
              {formatSignedNumber(analytics.realizedTotal)}
            </Title>
          </Space>
        </Card>

        <Card style={compactMetricCardStyle} styles={{ body: { padding: 14 } }}>
          <Space orientation="vertical" size={2} style={compactMetricStackStyle}>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>当日盈亏</Text>
            <Title
              level={4}
              style={{
                margin: 0,
                color:
                  strategy.todayRealizedPnl === undefined
                    ? "rgba(0, 0, 0, 0.88)"
                    : strategy.todayRealizedPnl > 0
                      ? "#15803d"
                      : strategy.todayRealizedPnl < 0
                        ? "#b91c1c"
                      : "rgba(0, 0, 0, 0.88)",
              }}
            >
              {strategy.todayRealizedPnl === undefined ? "--" : formatSignedNumber(strategy.todayRealizedPnl)}
            </Title>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>
              盈利笔数 {strategy.todayWinningJournalCount} / 亏损笔数 {strategy.todayLosingJournalCount}
            </Text>
          </Space>
        </Card>

        <Card style={compactMetricCardStyle} styles={{ body: { padding: 14 } }}>
          <Space orientation="vertical" size={2} style={compactMetricStackStyle}>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>总胜率</Text>
            <Title level={4} style={{ margin: 0, color: "rgba(0, 0, 0, 0.88)" }}>
              {strategy.winRate === undefined ? "--" : `${formatPercent(strategy.winRate)}%`}
            </Title>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>
              {strategy.completedJournalCount > 0
                ? `盈利 ${strategy.winningJournalCount} / 总计 ${strategy.completedJournalCount}`
                : "暂无历史样本"}
            </Text>
          </Space>
        </Card>

        <Card style={compactMetricCardStyle} styles={{ body: { padding: 14 } }}>
          <Space orientation="vertical" size={2} style={compactMetricStackStyle}>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>总盈亏比</Text>
            <Title level={4} style={{ margin: 0, color: "rgba(0, 0, 0, 0.88)" }}>
              {strategy.profitLossRatio === undefined ? "--" : formatRatio(strategy.profitLossRatio)}
            </Title>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>
              {strategy.completedJournalCount === 0
                ? "暂无历史样本"
                : strategy.averageWin === undefined
                  ? "暂无盈利样本"
                  : strategy.averageLoss === undefined
                    ? "暂无亏损样本"
                    : `平均盈利 ${formatSignedNumber(strategy.averageWin)} / 平均亏损 -${formatNumber(strategy.averageLoss)}`}
            </Text>
          </Space>
        </Card>

        <Card style={compactMetricCardStyle} styles={{ body: { padding: 14 } }}>
          <Space orientation="vertical" size={2} style={compactMetricStackStyle}>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>已统计 Journal</Text>
            <Title level={4} style={{ margin: 0, color: "rgba(0, 0, 0, 0.88)" }}>
              {analytics.journalCount}
            </Title>
          </Space>
        </Card>
      </div>

      <div style={topAnalysisGridStyle}>
        <Card style={{ ...shellCardStyle, height: "100%" }}>
          <Space orientation="vertical" size={12} style={{ width: "100%" }}>
            {renderCardTitle("盈亏趋势", "按平仓事件累计，不包含未平仓浮盈浮亏。")}
            <Line {...lineConfig} />
          </Space>
        </Card>

        <Card style={{ ...shellCardStyle, height: "100%" }}>
          <Space orientation="vertical" size={12} style={{ width: "100%" }}>
            {renderCardTitle(
              "最近一年盈亏热力图",
              "按 UTC 日期展示近一年已实现盈亏，绿色越深代表盈利越高，红色越深代表亏损越大。",
            )}

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
      </div>

      <div style={smallCardGridStyle}>
        <Card style={{ ...shellCardStyle, gridColumn: "1 / -1" }}>
          <Space orientation="vertical" size={12} style={{ width: "100%" }}>
            {renderCardTitle("每日已实现盈亏", "按 UTC 日期聚合，盈利为绿色，亏损为红色。")}
            <Column {...columnConfig} />
          </Space>
        </Card>
      </div>
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

function renderCardTitle(title: string, description: string) {
  return (
    <div style={cardTitleRowStyle}>
      <Title level={5} style={cardTitleStyle}>{title}</Title>
      <Tooltip title={description}>
        <InfoCircleOutlined aria-label={`${title}说明`} style={cardDescriptionIconStyle} />
      </Tooltip>
    </div>
  );
}

function formatPercent(value: number): string {
  return value.toFixed(1).replace(/\.0$/, "");
}

function formatRatio(value: number): string {
  return value.toFixed(2).replace(/\.00$/, "");
}

function formatNumber(value: number): string {
  return new Intl.NumberFormat("en-US", {
    maximumFractionDigits: 2,
  }).format(value);
}
