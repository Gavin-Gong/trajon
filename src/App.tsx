import { Desktop } from "./pages/Desktop";
import { parseAppView } from "./platform/panelWindow";
import "./App.css";

function App() {
  const view = parseAppView(window.location.search);
  return <Desktop view={view} />;
}

export default App;
