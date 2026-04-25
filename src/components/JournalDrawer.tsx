import { Drawer } from "antd";

import { JournalWorkspace } from "./JournalWorkspace";
import type { TradingJournalStore } from "./JournalStoreTypes";

export function JournalDrawer({
  store,
  open,
  journalId,
  onClose,
}: {
  store: TradingJournalStore;
  open: boolean;
  journalId?: string;
  onClose: () => void;
}) {
  const journal = store.state.journals.find((item) => item.id === journalId);

  return (
    <Drawer
      title={journal ? `${journal.symbol || "未命名"} Journal` : "Journal"}
      placement="right"
      size="large"
      closable={{ placement: "end" }}
      open={open}
      onClose={onClose}
      destroyOnHidden
      styles={{ body: { padding: 0 } }}
    >
      <JournalWorkspace store={store} journal={journal} onAfterFinish={onClose} />
    </Drawer>
  );
}
