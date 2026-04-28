import { Flex, Typography } from "antd";

import { useReminderClock } from "../hooks/useReminderClock";
import { formatCalendarDate, formatClock } from "./JournalFormatters";

const { Text } = Typography;

const clockShellStyle = {
  minWidth: 108,
  textAlign: "center" as const,
};

export function PanelHeader({
  timerEnabled,
  timerInterval,
}: {
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
            display: "flex",
            alignItems: "baseline",
            justifyContent: "center",
            gap: 10,
            background: reminder.isFlashing ? "rgba(37, 99, 235, 0.12)" : "white",
            boxShadow: reminder.isFlashing ? "0 12px 24px rgba(37, 99, 235, 0.16)" : undefined,
          }}
        >
          <Text
            style={{
              display: "block",
              color: reminder.isFlashing ? "#2563eb" : "rgba(0, 0, 0, 0.88)",
              fontSize: 36,
              fontWeight: 700,
              lineHeight: 1.1,
              letterSpacing: "0.08em",
              fontVariantNumeric: "tabular-nums",
            }}
          >
            {formatClock(reminder.now)}
          </Text>

          <Text
            style={{
              color: reminder.isFlashing ? "#2563eb" : "rgba(0, 0, 0, 0.6)",
              fontSize: 14,
              fontWeight: 600,
              lineHeight: 1,
              fontVariantNumeric: "tabular-nums",
              whiteSpace: "nowrap",
            }}
          >
            {formatCalendarDate(reminder.now)}
          </Text>
        </div>

      </Flex>
    </div>
  );
}
