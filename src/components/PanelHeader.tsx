import { PushpinFilled, PushpinOutlined } from "@ant-design/icons";
import { Button, Flex, Statistic } from "antd";

import { useReminderClock } from "../hooks/useReminderClock";
import { formatClock } from "./JournalFormatters";

const clockShellStyle = {
  minWidth: 108,
  textAlign: "center" as const,
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
    <div style={{ padding: 16, paddingBottom: 0 }}>
      <Flex align="center" justify="space-between" gap={12}>
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
                fontSize: 36,
                fontWeight: 700,
                letterSpacing: "0.08em",
                fontVariantNumeric: "tabular-nums",
                padding: 0
              },
            }}
          />
        </div>

        <Button
          type={isPinned ? "primary" : "default"}
          shape="circle"
          icon={isPinned ? <PushpinFilled /> : <PushpinOutlined />}
          title={isPinned ? "当前保持置顶" : "当前允许被覆盖"}
          onClick={onTogglePinned}
        />
      </Flex>
    </div>
  );
}
