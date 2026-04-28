import { Modal } from "antd";

import type { TradeJournal } from "../domain/journal";
import { ReviewTradingStep } from "./ReviewTradingStep";
import type { TradingJournalStore } from "./JournalStoreTypes";

export function ReviewTradingModal({
  open,
  journal,
  store,
  onClose,
  onConfirm,
}: {
  open: boolean;
  journal: TradeJournal | undefined;
  store: TradingJournalStore;
  onClose: () => void;
  onConfirm: (summary: string, tags: string[]) => void;
}) {
  return (
    <Modal title="复盘" open={open} destroyOnHidden footer={null} width={720} onCancel={onClose}>
      <ReviewTradingStep
        journal={journal}
        store={store}
        summary={journal?.summary ?? ""}
        tags={journal?.tags ?? []}
        onSummaryChange={() => undefined}
        onTagsChange={() => undefined}
        onBack={onClose}
        onConfirm={() => {
          onConfirm(journal?.summary ?? "", journal?.tags ?? []);
        }}
      />
    </Modal>
  );
}
