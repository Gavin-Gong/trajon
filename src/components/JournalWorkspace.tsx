import { message } from "antd";
import { useState } from "react";

import type { TradeJournal } from "../domain/journal";
import { PanelJournalSummary } from "./PanelJournalSummary";
import { QuickCaptureCard } from "./QuickCaptureCard";
import { ReviewTradingModal } from "./ReviewTradingModal";
import type { TradingJournalStore } from "./JournalStoreTypes";
import { TimelineSection } from "./TimelineSection";

const REVIEW_SAVE_SUCCESS_DURATION_SECONDS = 2;

export function JournalWorkspace({
  store,
  journal,
  onAfterFinish,
}: {
  store: TradingJournalStore;
  journal: TradeJournal | undefined;
  onAfterFinish?: () => Promise<void> | void;
}) {
  const [showReviewModal, setShowReviewModal] = useState(false);

  return (
    <>
      <PanelJournalSummary
        journal={journal}
        journals={store.state.journals}
        onUpdateSymbol={(journalId, symbol) => store.actions.updateJournalSymbol(journalId, symbol)}
        onFinish={() => {
          if (!journal?.symbol.trim()) {
            void message.warning("请先选择或填写交易标的");
            return;
          }

          setShowReviewModal(true);
        }}
      />

      <TimelineSection journal={journal} title="最近记录" />

      <QuickCaptureCard
        journal={journal}
        onRecord={(draft) =>
          journal
            ? store.actions.recordEvent(journal.id, draft)
            : Promise.resolve({ ok: false, error: "missingContent" } as const)
        }
      />

      <ReviewTradingModal
        open={showReviewModal && journal !== undefined}
        journal={journal}
        store={store}
        onClose={() => setShowReviewModal(false)}
        onConfirm={async (summary, tags) => {
          if (!journal) {
            return;
          }

          await store.actions.finishJournal(journal.id, summary, tags);
          message.success("保存成功", REVIEW_SAVE_SUCCESS_DURATION_SECONDS);
          setShowReviewModal(false);
          await new Promise((resolve) => {
            window.setTimeout(resolve, REVIEW_SAVE_SUCCESS_DURATION_SECONDS * 1000);
          });
          await onAfterFinish?.();
        }}
      />
    </>
  );
}
