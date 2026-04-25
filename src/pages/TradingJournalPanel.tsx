import { Flex } from "antd";
import { useEffect, useState } from "react";

import { JournalWorkspace } from "../components/JournalWorkspace";
import { PanelHeader } from "../components/PanelHeader";
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
  const { state } = store;
  const [focusedJournalId, setFocusedJournalId] = useState<string | undefined>(journalId);
  const { journal, targetPresetSymbol } = usePanelNavigationTarget(store, presetSymbol, focusedJournalId);
  const { isPinned, togglePinned } = usePanelPinnedState(store);

  useEffect(() => {
    setFocusedJournalId(journalId);
  }, [journalId]);

  useEffect(() => {
    if (journal?.id) {
      setFocusedJournalId(journal.id);
    }
  }, [journal?.id]);

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

        <JournalWorkspace
          store={store}
          journal={journal}
          onAfterFinish={async () => {
            await closePanelWindow();
          }}
        />
      </Flex>
    </main>
  );
}
