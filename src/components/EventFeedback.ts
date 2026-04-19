import { composeTradeEvent, type TradeEventComposerError, type TradeEventType, type TradeJournal } from "../domain/journal";

export function feedbackMessage(
  selectedType: TradeEventType,
  journal: TradeJournal,
  validation: ReturnType<typeof composeTradeEvent>,
  canSubmit: boolean,
) {
  if (journal.status === "done") {
    return "已完成的 journal 现在只接受补充笔记。";
  }

  if (!canSubmit) {
    return validation.ok
      ? "补全必要字段后即可记录。"
      : validationMessage(selectedType, validation.error);
  }

  if (selectedType === "exit") {
    return "可以记录平仓。";
  }

  if (selectedType === "log") {
    return "可以记录观察。";
  }

  return "可以记录事件。";
}

function validationMessage(type: TradeEventType, error: TradeEventComposerError) {
  switch (error) {
    case "missingContent":
      return type === "log" ? "观察记录需要填写文本。" : "请先补全必要字段。";
    case "missingQuantity":
      return type === "entry" ? "开仓需要填写带方向的数量。" : "请先填写数量。";
    case "invalidPrecision":
      return "数量和价格最多保留两位小数。";
    case "zeroQuantity":
      return "数量不能为 0。";
    case "wrongDirection":
      return type === "add" ? "加仓方向必须与当前仓位一致。" : "减仓方向必须朝 0 回补。";
    case "exceedsPosition":
      return "减仓数量不能超过剩余仓位。";
    case "doneOnlyLogs":
      return "已完成的 journal 现在只允许补充笔记。";
  }
}

export function quantityHint(
  type: TradeEventType,
  status: TradeJournal["status"],
  currentNetPosition: number,
) {
  if (status === "done") {
    return "已完成的 journal 现在只允许补充笔记。";
  }

  switch (type) {
    case "entry":
      return "正数表示做多，负数表示做空；最多两位小数。";
    case "add":
      return currentNetPosition < 0
        ? "当前是空头仓位，加仓数量也必须保持负数。"
        : "当前是多头仓位，加仓数量也必须保持正数。";
    case "reduce":
      return currentNetPosition < 0
        ? "空头减仓时，数量应填写正数。"
        : "多头减仓时，数量应填写负数。";
    case "exit":
      return "平仓会自动使用全部剩余仓位。";
    case "log":
      return "";
  }
}
