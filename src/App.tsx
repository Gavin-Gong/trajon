import { parseAppView } from "./panelWindow";
import { TradingPanelDesktop } from "./tradingPanelApp";
import "./App.css";

function App() {
  const view = parseAppView(window.location.search);
  return <TradingPanelDesktop view={view} />;
}

export default App;
