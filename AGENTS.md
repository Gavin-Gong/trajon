# AGENTS.md

## Project overview

Trajon is a Go desktop market-data and candlestick-chart application. It uses `github.com/egoist/mygo` for the native UI and `github.com/oneo-me/onevra/source` for shared UI components and icons.

The module currently targets Go 1.27.1. Keep dependency versions pinned in `go.mod` unless an upgrade is explicitly requested.

## Repository layout

- `main.go` — process entry point; delegates to `internal/app`.
- `internal/app/` — application state, MyGo views, market-page orchestration, instrument selection, candles, and chart UI state.
- `internal/market/` — market-domain types, source interfaces, registry/loader behavior, indicators, and source-independent logic.
- `internal/market/adapters/` — exchange-specific candle-source implementations (currently Binance and OKX).
- `internal/market/defaults/` — default source registry and instrument catalog.
- `internal/chart/` — chart viewport calculations and other rendering-independent chart behavior.
- `internal/ui/` — small reusable UI helpers.

Keep exchange/API concerns in adapters, domain behavior in `internal/market`, and presentation/state orchestration in `internal/app` or `internal/ui`.

## Development commands

Run from the repository root:

```sh
gofmt -w .
go test ./...
CGO_ENABLED=0 go test ./...
go run github.com/egoist/mygo/cmd/mygo vet .
```

Use targeted package tests while iterating, for example `go test ./internal/market/...` or `go test ./internal/app`. Run the full suite before finishing. Add or update focused tests for changes to loaders, pagination, cache keys, instrument identity, indicators, adapters, or viewport calculations.

## Go and domain conventions

- Follow standard Go formatting and error handling. Prefer small, deterministic functions and explicit dependencies.
- Preserve the identity semantics of `market.Instrument.Identity()` and the request/cache semantics of `market.CandleRequest.CacheKey()` when changing market models.
- Implement new data providers through `market.CandleSource` and register them through the market registry; do not put provider-specific branching in UI code.
- Keep candle timestamps, ordering, intervals, pagination boundaries, and provider-symbol conversions covered by tests.
- Avoid mutating shared candle slices unexpectedly. Use the existing clone helpers where ownership crosses a boundary.
- Keep network and other slow work off the UI thread. Reject stale market-load results when the selected instrument/timeframe changes, and close loaders when the window closes.

## MyGo UI rules

- Store application data, widget state, and `ui.Handle` values—not transient `ui.Element` or parts that expire after a build pass.
- Give stateful controls stable, unique keys before construction and preserve identity across rebuilds.
- Use `Window.Update`/the established update path to publish background results; do not mutate UI state directly from worker goroutines.
- Keep drawing callbacks limited to painting and use existing MyGo/onevra components for controls, focus, keyboard behavior, and accessibility.
- Verify UI lifecycle and focus changes with the project’s existing tester patterns where applicable. Consult the pinned MyGo source before relying on an API.
- Replay-start selection previews all candles under a pointer-following gray future mask. Dragging only pans; an undragged chart click confirms the start, exits selection, and removes the selection marker. Confirmed replay still hides future candles until they are played.
- The floating action bar shows replay-start selection before a start is confirmed. After confirmation it shows play/pause, reset, and end instead. Replay settings (initial capital and leverage) appear only alongside replay-start selection before a start is confirmed; they stay hidden after start confirmation, including while playing, paused, reset, or completed, and return alongside replay-start selection after End. Settings use a save/cancel dialog and persist across replay sessions and instrument/timeframe changes within the application. End clears the replay session and restores all candles at the latest viewport position while preserving zoom and annotations.
- After replay-start confirmation, the floating action bar also shows a compact order-quantity input and TrendingUp/TrendingDown buy/sell buttons. Quantity is in units of the instrument. Trades fill at `candles[replayEnd-1].Close` and its timestamp, independently of hover/viewport, without pausing playback. A single net position supports longs and shorts; opposing orders close first and reverse any remainder. Initial capital plus realized/unrealized P&L and leverage limit opening exposure; closing remains possible under a margin deficit. No fees, slippage, or automatic liquidation are modeled. Completed replay disables trading. Reset, End, a new replay start, and a committed instrument/timeframe change clear positions and fills; saved replay settings remain intact.
- Non-flat replay positions show a horizontal average-entry line with a direction/quantity tag and a native close button; closing uses the latest revealed candle's Close for the entire position, not the order input. Hovering/dragging the line creates a take-profit or stop-loss preview (long: up=profit/down=loss; short: reversed); release commits and Esc cancels. Existing brackets can be dragged or canceled individually. Price projection is frozen during a drag, and the chart range includes positions and active brackets. Brackets apply to the whole remaining position, survive same-direction additions/reductions, and clear on flat/reversal or session reset. Only candles after bracket creation/modification can trigger: High/Low touches fill at the line, gaps at Open, and same-candle dual touches prioritize stop-loss. Check exits before completing the final replay candle; successful exits do not pause playback. Completed replay leaves any unclosed position visible but disables trading and bracket edits.

## Change hygiene

- Keep user-facing copy concise and consistent with the existing Chinese UI terminology; do not add decorative or marketing copy.
- Do not commit generated artifacts, credentials, or local build output.
- Prefer a focused change over broad refactors. If behavior or public project conventions change, update this file and the relevant tests together.
