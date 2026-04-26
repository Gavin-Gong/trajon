import { AutoComplete, Button, Card, Descriptions, Flex, Modal, Select, Space, Switch, Tag, Typography, message } from "antd";
import { useEffect, useState } from "react";

import { TIMEFRAME_PRESET_VALUES, type TimeframePreset } from "../model/tradingPairPreset";
import {
  buildDatabaseBackupFileName,
  createDatabaseBackup,
  isDatabaseBackupAvailable,
  listDatabaseBackups,
  normalizeDatabaseBackupFileName,
  restoreDatabaseBackup,
  type DatabaseBackupFile,
} from "../platform/dataBackup";
import { tagColorForVariant } from "./EventPresentation";
import { REMINDER_INTERVALS } from "./JournalFormatters";
import { mutedShellCardStyle, panelCardStyle, shellCardStyle } from "./PanelStyles";
import type { TradingJournalStore } from "./JournalStoreTypes";

const { Paragraph, Text, Title } = Typography;

export function SettingsSection({ store }: { store: TradingJournalStore }) {
  const { config } = store.state;
  const [presetSymbolInput, setPresetSymbolInput] = useState("");
  const [presetTimeframe, setPresetTimeframe] = useState<TimeframePreset>("5m");
  const [backupFiles, setBackupFiles] = useState<DatabaseBackupFile[]>([]);
  const [createBackupModalOpen, setCreateBackupModalOpen] = useState(false);
  const [backupFileNameInput, setBackupFileNameInput] = useState(buildDatabaseBackupFileName());
  const [backupModalOpen, setBackupModalOpen] = useState(false);
  const [selectedBackupFileName, setSelectedBackupFileName] = useState<string>();
  const [backupPending, setBackupPending] = useState(false);
  const [restorePending, setRestorePending] = useState(false);

  const databaseBackupAvailable = isDatabaseBackupAvailable();

  const presetOptions = store.state.pairPresets.map((preset) => ({
    value: preset.symbol,
    label: preset.symbol,
  }));

  const canSavePreset = presetSymbolInput.trim().length > 0;

  useEffect(() => {
    if (!databaseBackupAvailable) {
      setBackupFiles([]);
      return;
    }

    void listDatabaseBackups()
      .then((nextBackups) => {
        setBackupFiles(nextBackups);
      })
      .catch(() => {
        setBackupFiles([]);
      });
  }, [databaseBackupAvailable]);

  function handleOpenCreateBackupModal() {
    setBackupFileNameInput(buildDatabaseBackupFileName());
    setCreateBackupModalOpen(true);
  }

  async function handleCreateBackup() {
    const fileName = normalizeDatabaseBackupFileName(backupFileNameInput);
    if (!fileName) {
      message.error("请输入备份文件名");
      return;
    }

    setBackupPending(true);

    try {
      const backup = await createDatabaseBackup(fileName);
      message.success(`已创建备份：${backup.fileName}`);
      const nextBackups = await listDatabaseBackups();
      setBackupFiles(nextBackups);
      setCreateBackupModalOpen(false);
    } catch (error) {
      message.error(error instanceof Error ? error.message : "创建备份失败");
    } finally {
      setBackupPending(false);
    }
  }

  async function handleOpenRestoreModal() {
    setRestorePending(true);
    setBackupModalOpen(true);

    try {
      const nextBackups = await listDatabaseBackups();
      setBackupFiles(nextBackups);
      setSelectedBackupFileName(nextBackups[0]?.fileName);
    } catch (error) {
      setBackupModalOpen(false);
      message.error(error instanceof Error ? error.message : "读取备份列表失败");
    } finally {
      setRestorePending(false);
    }
  }

  async function handleConfirmRestore() {
    if (!selectedBackupFileName) {
      return;
    }

    Modal.confirm({
      title: "确认恢复数据库备份",
      content: "恢复会整库覆盖当前数据，当前 journal、预设和设置将被替换。",
      okText: "确认覆盖并恢复",
      cancelText: "取消",
      okButtonProps: { danger: true },
      async onOk() {
        setRestorePending(true);

        try {
          await restoreDatabaseBackup(selectedBackupFileName);
          await store.actions.refreshState();
          setBackupModalOpen(false);
          message.success(`已恢复备份：${selectedBackupFileName}`);
        } catch (error) {
          message.error(error instanceof Error ? error.message : "恢复备份失败");
        } finally {
          setRestorePending(false);
        }
      },
    });
  }

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
              数据库备份与恢复
            </Title>
            <Paragraph style={{ margin: "4px 0 0", color: "rgba(0, 0, 0, 0.45)" }}>
              当前版本支持桌面端 SQLite 数据库备份与整库恢复，恢复会直接覆盖当前数据。
            </Paragraph>
          </div>

          <Descriptions
            size="small"
            column={1}
            items={[
              { key: "storage", label: "数据位置", children: "本地 SQLite" },
              { key: "backup", label: "备份位置", children: "~/trajon/backups" },
              { key: "restore", label: "恢复策略", children: "整库覆盖" },
            ]}
          />

          <Space orientation="vertical" size={12} style={{ width: "100%" }}>
            <Button block type="primary" loading={backupPending} disabled={!databaseBackupAvailable} onClick={handleOpenCreateBackupModal}>
              创建备份
            </Button>
            <Button
              block
              disabled={!databaseBackupAvailable || backupFiles.length === 0}
              loading={restorePending}
              onClick={() => void handleOpenRestoreModal()}
            >
              恢复备份
            </Button>
          </Space>

          <Modal
            title="创建数据库备份"
            open={createBackupModalOpen}
            okText="创建备份"
            cancelText="取消"
            confirmLoading={backupPending}
            onOk={() => void handleCreateBackup()}
            onCancel={() => {
              if (backupPending) {
                return;
              }

              setCreateBackupModalOpen(false);
            }}
          >
            <Space orientation="vertical" size={12} style={{ width: "100%" }}>
              <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>输入备份文件名，未填写 `.db` 时会自动补齐。</Text>
              <Space orientation="vertical" size={8} style={{ width: "100%" }}>
                <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>备份文件名</Text>
                <AutoComplete
                  value={backupFileNameInput}
                  options={[]}
                  onChange={(value: string) => setBackupFileNameInput(value)}
                  placeholder="例如 trajon-backup-20260426-090807.db"
                  style={{ width: "100%" }}
                />
              </Space>
            </Space>
          </Modal>

          {backupModalOpen ? (
            <Card style={panelCardStyle}>
              <Space orientation="vertical" size={12} style={{ width: "100%" }}>
                <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>恢复数据库备份</Text>
                <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>选择一个内部备份文件恢复当前数据库。</Text>
                <Space orientation="vertical" size={8} style={{ width: "100%" }}>
                  <Text strong style={{ color: "rgba(0, 0, 0, 0.88)" }}>备份文件</Text>
                  <Select
                    aria-label="选择备份文件"
                    value={selectedBackupFileName}
                    placeholder="选择一个备份文件"
                    options={backupFiles.map((backup) => ({ value: backup.fileName, label: backup.fileName }))}
                    onChange={(value) => setSelectedBackupFileName(value)}
                    style={{width: "100%"}}
                    popupMatchSelectWidth={270}
                  />
                </Space>
                <Text style={{ color: "#b91c1c" }}>恢复会整库覆盖当前数据，请谨慎操作。</Text>
                <Flex justify="end" gap={8}>
                  <Button
                    onClick={() => {
                      if (restorePending) {
                        return;
                      }

                      setBackupModalOpen(false);
                    }}
                  >
                    取消
                  </Button>
                  <Button
                    type="primary"
                    danger
                    loading={restorePending}
                    disabled={!selectedBackupFileName}
                    onClick={() => void handleConfirmRestore()}
                  >
                    确认恢复
                  </Button>
                </Flex>
              </Space>
            </Card>
          ) : null}
        </Space>
      </Card>
    </div>
  );
}
