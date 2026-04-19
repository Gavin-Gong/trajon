import type { AppView } from "../components/AppTypes";
import { useTradingJournalState } from "../stores/useTradingJournalState";
import { Main } from "./Main";
import { TradingJournalPanel } from "./TradingJournalPanel";

export function Desktop({ view }: { view: AppView }) {
  const store = useTradingJournalState();

  if (view.view === "panel") {
    return <TradingJournalPanel store={store} presetSymbol={view.presetSymbol} journalId={view.journalId} />;
  }

  return <Main store={store} />;
}
