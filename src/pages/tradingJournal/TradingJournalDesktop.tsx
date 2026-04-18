import { TradingJournalMain } from "./TradingJournalMain";
import { TradingJournalPanel } from "./TradingJournalPanel";
import type { AppView } from "../../components/tradingJournal/TradingJournalTypes";
import { useTradingJournalState } from "../../stores/useTradingJournalState";

export function TradingJournalDesktop({ view }: { view: AppView }) {
  const store = useTradingJournalState();

  if (view.view === "panel") {
    return <TradingJournalPanel store={store} presetSymbol={view.presetSymbol} />;
  }

  return <TradingJournalMain store={store} />;
}
