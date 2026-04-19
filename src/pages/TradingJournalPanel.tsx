import { Flex } from "antd";
import { useState } from "react";

import {  FinishJournalModal } from "../components/FinishJournalModal";
import { PanelHeader } from "../components/PanelHeader";
import { PanelJournalSummary } from "../components/PanelJournalSummary";
import { QuickCaptureCard } from "../components/QuickCaptureCard";
import { TimelineSection } from "../components/TimelineSection";
import type { TradingJournalStore } from "../components/JournalStoreTypes";
import { usePanelAutoCreateJournal } from "../hooks/usePanelAutoCreateJournal";
import { usePanelNavigationTarget } from "../hooks/usePanelNavigationTarget";
import { usePanelPinnedState } from "../hooks/usePanelPinnedState";
import { usePersistPanelBounds } from "../hooks/usePersistPanelBounds";
import { closePanelWindow } from "../platform/panelLifecycle";

const panelShellStyle = {
  margin: "0 auto",
  width: "100%",
  maxWidth: 430,
};

export function TradingJournalPanel({
  store,
  presetSymbol,
  journalId,
}: {
  store: TradingJournalStore;
  presetSymbol?: string;
  journalId?: string;
}) {
  const { state, actions } = store;
  const { journal, targetPresetSymbol } = usePanelNavigationTarget(store, presetSymbol, journalId);
  const [showFinishJournal, setShowFinishJournal] = useState(false);
  const { isPinned, togglePinned } = usePanelPinnedState(store);

  usePersistPanelBounds();
  usePanelAutoCreateJournal(store, journal, targetPresetSymbol);

  return (
    <main style={{ minHeight: "100vh" }}>
      <Flex vertical gap={16} style={panelShellStyle}>
        <PanelHeader
          isPinned={isPinned}
          onTogglePinned={togglePinned}
          timerEnabled={state.config.timerEnabled}
          timerInterval={state.config.timerInterval}
        />

        <PanelJournalSummary journal={journal} onFinish={() => setShowFinishJournal(true)} />

        <TimelineSection journal={journal} title="最近记录" />

        <QuickCaptureCard
          journal={journal}
          onRecord={(draft) =>
            journal
              ? store.actions.recordEvent(journal.id, draft)
              : Promise.resolve({ ok: false, error: "missingContent" } as const)
          }
        />
      </Flex>

      <FinishJournalModal
        open={showFinishJournal && journal !== undefined}
        initialSummary={journal?.summary ?? ""}
        onClose={() => setShowFinishJournal(false)}
        onConfirm={(summary) => {
          if (!journal) {
            return;
          }

          actions.finishJournal(journal.id, summary);
          setShowFinishJournal(false);
          void closePanelWindow();
        }}
      />
    </main>
  );
}
