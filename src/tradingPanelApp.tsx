import { MainSurface } from "./components/tradingPanel/TradingPanelMain";
import { PanelSurface } from "./components/tradingPanel/TradingPanelPanel";
import type { AppView } from "./components/tradingPanel/TradingPanelTypes";
import { useTradingPanelState } from "./tradingPanelStore";

export function TradingPanelDesktop({ view }: { view: AppView }) {
  const store = useTradingPanelState();

  if (view === "panel") {
    return <PanelSurface store={store} />;
  }

  return <MainSurface store={store} />;
}
