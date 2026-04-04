import { FormEvent, useState } from "react";

import { openDetailWindow, parseViewState } from "./detailWindow";
import "./App.css";

function App() {
  const viewState = parseViewState(window.location.search);
  const [symbol, setSymbol] = useState("");
  const [error, setError] = useState("");

  async function handleOpenDetailWindow(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const trimmedSymbol = symbol.trim();
    if (!trimmedSymbol) {
      setError("Enter a symbol to open a detail window.");
      return;
    }

    setError("");
    await openDetailWindow(trimmedSymbol);
  }

  if (viewState.view === "detail") {
    return (
      <main className="detail-shell">
        <section className="detail-card">
          <p className="eyebrow">Independent Tauri Window</p>
          <h1>{viewState.symbol}</h1>
          <p className="detail-copy">
            This detail view was opened from the main window and can stay open
            alongside other symbols.
          </p>
          <dl className="detail-meta">
            <div>
              <dt>Symbol</dt>
              <dd>{viewState.symbol}</dd>
            </div>
            <div>
              <dt>Query</dt>
              <dd>{window.location.search}</dd>
            </div>
          </dl>
        </section>
      </main>
    );
  }

  return (
    <main className="app-shell">
      <section className="launcher-card">
        <p className="eyebrow">Tauri Multi-Window Demo</p>
        <h1>Open Symbol Detail Windows</h1>
        <p className="lead">
          Each symbol gets one dedicated window. Opening the same symbol again
          will focus the existing window instead of creating a duplicate.
        </p>

        <form className="launcher-form" onSubmit={handleOpenDetailWindow}>
          <label className="field-label" htmlFor="symbol-input">
            Symbol
          </label>
          <div className="input-row">
            <input
              id="symbol-input"
              value={symbol}
              onChange={(event) => setSymbol(event.currentTarget.value)}
              placeholder="BTCUSDT"
            />
            <button type="submit">Open Detail</button>
          </div>
        </form>

        <p className="hint">Try `BTCUSDT`, `ETHUSDT`, or `SOL/USDT`.</p>
        {error ? <p className="error-text">{error}</p> : null}
      </section>
    </main>
  );
}

export default App;
