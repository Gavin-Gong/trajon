import { AutoComplete, Button, Card, Descriptions, Flex, Select, Space, Switch, Tag, Typography } from "antd";
import { useState } from "react";

import { TIMEFRAME_PRESET_VALUES, type TimeframePreset } from "../model/tradingPairPreset";
import { tagColorForVariant } from "./EventPresentation";
import { REMINDER_INTERVALS } from "./JournalFormatters";
import { dashedShellCardStyle, mutedShellCardStyle, panelCardStyle, shellCardStyle } from "./PanelStyles";
import type { TradingJournalStore } from "./JournalStoreTypes";

const { Paragraph, Text, Title } = Typography;

export function SettingsSection({ store }: { store: TradingJournalStore }) {
  const { config } = store.state;
  const [presetSymbolInput, setPresetSymbolInput] = useState("");
  const [presetTimeframe, setPresetTimeframe] = useState<TimeframePreset>("5m");

  const presetOptions = store.state.pairPresets.map((preset) => ({
    value: preset.symbol,
    label: preset.symbol,
  }));

  const canSavePreset = presetSymbolInput.trim().length > 0;

  return (
    <div
      style={{
        display: "grid",
        gap: 24,
        gridTemplateColumns: "minmax(0, 1fr) 320px",
      }}
    >
      <Card style={shellCardStyle}>
        <Space orientation="vertical" size={16} style={{ width: "100%" }}>
          <Card style={mutedShellCardStyle}>
            <Flex justify="space-between" align="center" gap={16}>
              <Space orientation="vertical" size={4}>
                <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>启用提醒</Text>
                <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>按固定频率提醒自己记录盘中观察。</Text>
              </Space>
              <Switch
                checked={config.timerEnabled}
                onChange={(checked) => {
                  void store.actions.updateConfig({ timerEnabled: checked });
                }}
              />
            </Flex>
          </Card>

          <Card style={panelCardStyle}>
            <Space orientation="vertical" size={12} style={{ width: "100%" }}>
              <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>提醒间隔</Text>
              <Paragraph style={{ margin: 0, color: "rgba(0, 0, 0, 0.45)" }}>
                建议与自己的盘中节奏保持一致，避免提醒过密造成打断。
              </Paragraph>
              <Select
                value={config.timerInterval}
                disabled={!config.timerEnabled}
                options={REMINDER_INTERVALS.map((preset) => ({
                  value: preset.value,
                  label: preset.label,
                }))}
                onChange={(value) => {
                  void store.actions.updateConfig({ timerInterval: value });
                }}
              />
            </Space>
          </Card>

          <Card style={panelCardStyle}>
            <Space orientation="vertical" size={12} style={{ width: "100%" }}>
              <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>交易对预设</Text>
              <Paragraph style={{ margin: 0, color: "rgba(0, 0, 0, 0.45)" }}>
                为常用标的设置默认周期，并选择一个默认预设用于新建时自动带出。
              </Paragraph>

              <Flex gap={12} wrap="wrap" align="end">
                <Space orientation="vertical" size={8} style={{ minWidth: 220, flex: 1 }}>
                  <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>Symbol</Text>
                  <div data-testid="pair-preset-symbol-input">
                    <AutoComplete
                      value={presetSymbolInput}
                      virtual={false}
                      options={presetSymbolInput.trim().length === 0 ? [] : presetOptions}
                      placeholder="输入或选择交易对，例如 BTCUSDT"
                      filterOption={(inputValue: string, option?: { value?: string }) =>
                        (option?.value ?? "").toLowerCase().includes(inputValue.toLowerCase())
                      }
                      onChange={(value: string) => setPresetSymbolInput(value)}
                      style={{ width: "100%" }}
                    />
                  </div>
                </Space>

                <Space orientation="vertical" size={8} style={{ minWidth: 140 }}>
                  <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>Timeframe</Text>
                  <div data-testid="pair-preset-timeframe-select">
                    <Select
                      value={presetTimeframe}
                      virtual={false}
                      options={TIMEFRAME_PRESET_VALUES.map((value) => ({ value, label: value }))}
                      onChange={(value) => setPresetTimeframe(value as TimeframePreset)}
                      style={{ minWidth: 140 }}
                    />
                  </div>
                </Space>

                <Button
                  type="primary"
                  disabled={!canSavePreset}
                  onClick={() => {
                    if (!canSavePreset) {
                      return;
                    }

                    void store.actions.upsertPairPreset(presetSymbolInput, presetTimeframe);
                    setPresetSymbolInput("");
                  }}
                >
                  保存为预设
                </Button>
              </Flex>

              <Button
                type="default"
                onClick={() => {
                  void store.actions.upsertPairPreset("BTCUSDT", "5m");
                  void store.actions.setDefaultPresetSymbol("BTCUSDT");
                }}
                style={{ display: "none" }}
              >
                测试入口
              </Button>

              <Space data-testid="pair-preset-list" orientation="vertical" size={8} style={{ width: "100%" }}>
                <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>默认预设</Text>
                {store.state.pairPresets.length === 0 ? (
                  <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>还没有保存任何交易对预设。</Text>
                ) : (
                  store.state.pairPresets.map((preset) => {
                    const isDefault = store.state.defaultPresetSymbol === preset.symbol;
                    return (
                      <Flex key={preset.symbol} justify="space-between" align="center" gap={12}>
                        <Space align="center" size={8}>
                          <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>{preset.symbol}</Text>
                          <Tag color={tagColorForVariant("neutral")}>{preset.timeframe}</Tag>
                          {isDefault ? <Tag color={tagColorForVariant("accent")}>默认预设</Tag> : null}
                        </Space>
                        {isDefault ? null : (
                          <Button
                            size="small"
                            onClick={() => {
                              void store.actions.setDefaultPresetSymbol(preset.symbol);
                            }}
                            aria-label={`设为默认-${preset.symbol}`}
                          >
                            设为默认
                          </Button>
                        )}
                      </Flex>
                    );
                  })
                )}
              </Space>
            </Space>
          </Card>
        </Space>
      </Card>

      <Card style={{ ...shellCardStyle, height: "fit-content" }}>
        <Space orientation="vertical" size={16} style={{ width: "100%" }}>
          <div>
            <Title level={5} style={{ margin: 0, color: "rgba(0, 0, 0, 0.88)", letterSpacing: "-0.02em" }}>
              数据与导出
            </Title>
            <Paragraph style={{ margin: "4px 0 0", color: "rgba(0, 0, 0, 0.45)" }}>
              目前数据保存在本地。导出入口保留在这里，便于后续接入 CSV。
            </Paragraph>
          </div>

          <Descriptions
            size="small"
            column={1}
            items={[
              { key: "storage", label: "数据位置", children: "本地存储" },
              { key: "export", label: "导出状态", children: "即将支持" },
            ]}
          />

          <Card style={dashedShellCardStyle}>
            <Text style={{ color: "rgba(0, 0, 0, 0.45)", lineHeight: 1.8 }}>
              当前版本优先把记录链路和复盘体验打磨稳定，导出能力后续接入。
            </Text>
          </Card>

          <Button block onClick={() => undefined}>
            导出 CSV（即将支持）
          </Button>
        </Space>
      </Card>
    </div>
  );
}
