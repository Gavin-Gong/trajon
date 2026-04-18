import { TradingJournalDesktop } from "./pages/tradingJournal/TradingJournalDesktop";
import { parseAppView } from "./platform/panelWindow";
import "./App.css";

function App() {
  const view = parseAppView(window.location.search);
  return <TradingJournalDesktop view={view} />;
}

export default App;
