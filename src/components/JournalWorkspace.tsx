import { PushpinOutlined } from "@ant-design/icons";
import { Button, Flex, message, Segmented, Space, Typography } from "antd";
import { useEffect, useState } from "react";

import { netPosition, type TradeJournal } from "../domain/journal";
import { PanelJournalSummary } from "./PanelJournalSummary";
import { QuickCaptureCard } from "./QuickCaptureCard";
import { ReviewTradingStep } from "./ReviewTradingStep";
import type { TradingJournalStore } from "./JournalStoreTypes";
import { TimelineSection } from "./TimelineSection";

const REVIEW_SAVE_SUCCESS_DURATION_SECONDS = 2;
const { Text } = Typography;

type WorkspaceTab = "capture" | "review";

export function JournalWorkspace({
  store,
  journal,
  isPinned,
  onTogglePinned,
  onAfterFinish,
}: {
  store: TradingJournalStore;
  journal: TradeJournal | undefined;
  isPinned: boolean;
  onTogglePinned: () => void;
  onAfterFinish?: () => Promise<void> | void;
}) {
  const [currentTab, setCurrentTab] = useState<WorkspaceTab>("capture");
  const [reviewSummary, setReviewSummary] = useState(journal?.summary ?? "");
  const [reviewTags, setReviewTags] = useState<string[]>(journal?.tags ?? []);
  const hasRecordedEvents = (journal?.events.length ?? 0) > 0;
  const isNetPositionFlat = journal ? netPosition(journal.events) === 0 : false;
  const canOpenReview = hasRecordedEvents && isNetPositionFlat;

  useEffect(() => {
    setReviewSummary(journal?.summary ?? "");
    setReviewTags(journal?.tags ?? []);
    setCurrentTab("capture");
  }, [journal?.id, journal?.summary, journal?.tags]);

  const reviewHint = !hasRecordedEvents
    ? "至少先记录一条交易事件"
    : !isNetPositionFlat
      ? "请先把净仓归零后再复盘"
      : undefined;

  const handleTabChange = (nextTab: WorkspaceTab) => {
    if (nextTab === "capture") {
      setCurrentTab("capture");
      return;
    }

    if (!journal?.symbol.trim()) {
      void message.warning("请先选择或填写交易标的");
      return;
    }

    if (!canOpenReview) {
      if (reviewHint) {
        void message.warning(reviewHint);
      }
      return;
    }

    setCurrentTab("review");
  };

  return (
    <Space orientation="vertical" size={12} style={{ width: "100%" }}>
      <div style={{ padding: "0 12px" }}>
        <Flex align="center" justify="space-between" gap={12}>
          <div
            style={{
              display: "inline-flex",
              padding: 4,
              borderRadius: 999,
              background: "rgba(15, 23, 42, 0.06)",
            }}
          >
            <Segmented<WorkspaceTab>
              value={currentTab}
              options={[
                { label: "记录交易", value: "capture" },
                { label: "复盘总结", value: "review" },
              ]}
              onChange={(value) => {
                handleTabChange(value as WorkspaceTab);
              }}
            />
          </div>

          <Button
            type="default"
            shape="circle"
            style={{
              borderColor: isPinned ? "#27272a" : undefined,
            }}
            icon={
              <PushpinOutlined
                style={{
                  transform: isPinned ? "rotate(-45deg)" : undefined,
                  transition: "transform 0.2s ease",
                }}
              />
            }
            title={isPinned ? "当前保持置顶" : "当前允许被覆盖"}
            onClick={onTogglePinned}
          />
        </Flex>
      </div>

      {currentTab === "capture" ? (
        <>
          <PanelJournalSummary
            journal={journal}
            journals={store.state.journals}
            onUpdateSymbol={(journalId, symbol) => store.actions.updateJournalSymbol(journalId, symbol)}
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
        </>
      ) : (
        <div style={{ padding: "0 12px 12px" }}>
          <ReviewTradingStep
            journal={journal}
            store={store}
            summary={reviewSummary}
            tags={reviewTags}
            onSummaryChange={setReviewSummary}
            onTagsChange={setReviewTags}
            onBack={() => setCurrentTab("capture")}
            onConfirm={async () => {
              if (!journal) {
                return;
              }

              await store.actions.finishJournal(journal.id, reviewSummary, reviewTags);
              message.success("保存成功", REVIEW_SAVE_SUCCESS_DURATION_SECONDS);
              setCurrentTab("capture");
              await new Promise((resolve) => {
                window.setTimeout(resolve, REVIEW_SAVE_SUCCESS_DURATION_SECONDS * 1000);
              });
              await onAfterFinish?.();
            }}
          />
        </div>
      )}
    </Space>
  );
}
