import { InfoCircleOutlined, PushpinFilled, PushpinOutlined } from "@ant-design/icons";
import { Button, Flex, Space, Statistic, Tooltip, Typography } from "antd";

import { useReminderClock } from "../hooks/useReminderClock";
import { formatClock, formatReminderStatus } from "./JournalFormatters";
import { plainPanelSectionStyle } from "./PanelStyles";

const { Text } = Typography;

const clockShellStyle = {
  minWidth: 108,
  padding: "10px 12px",
  textAlign: "center" as const,
};

const infoIconStyle = {
  color: "rgba(0, 0, 0, 0.45)",
  cursor: "help",
  fontSize: 14,
};

export function PanelHeader({
  isPinned,
  onTogglePinned,
  timerEnabled,
  timerInterval,
}: {
  isPinned: boolean;
  onTogglePinned: () => void;
  timerEnabled: boolean;
  timerInterval: number;
}) {
  const reminder = useReminderClock(timerEnabled, timerInterval);

  return (
    <div style={plainPanelSectionStyle}>
      <Flex align="center" justify="space-between" gap={12}>
        <Button
          type={isPinned ? "primary" : "default"}
          shape="circle"
          icon={isPinned ? <PushpinFilled /> : <PushpinOutlined />}
          title={isPinned ? "当前保持置顶" : "当前允许被覆盖"}
          onClick={onTogglePinned}
        />

        <Space orientation="vertical" size={2} style={{ flex: 1, minWidth: 0 }}>
          <Text
            strong
            style={{
              fontSize: 11,
              letterSpacing: "0.18em",
              color: "rgba(0, 0, 0, 0.65)",
              textTransform: "uppercase",
            }}
          >
            Quick Capture Panel
          </Text>
          <Flex align="center" gap={6} wrap="wrap">
            <Text style={{ color: "rgba(0, 0, 0, 0.45)" }}>
              提醒 {formatReminderStatus(timerEnabled, timerInterval)}
            </Text>
            <Tooltip title="提醒仅用于盘中节奏检查，不会替代交易记录。">
              <InfoCircleOutlined aria-label="提醒说明" style={infoIconStyle} />
            </Tooltip>
          </Flex>
        </Space>

        <div
          style={{
            ...clockShellStyle,
            background: reminder.isFlashing ? "rgba(37, 99, 235, 0.12)" : "white",
            boxShadow: reminder.isFlashing ? "0 12px 24px rgba(37, 99, 235, 0.16)" : undefined,
          }}
        >
          <Statistic
            value={formatClock(reminder.now)}
            styles={{
              content: {
                color: reminder.isFlashing ? "#2563eb" : "rgba(0, 0, 0, 0.88)",
                fontSize: 15,
                fontWeight: 700,
                letterSpacing: "0.08em",
                fontVariantNumeric: "tabular-nums",
              },
            }}
          />
        </div>
      </Flex>
    </div>
  );
}
