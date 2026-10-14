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
- `internal/locales/` — typed Simplified Chinese and English UI catalogs, system-locale matching, and language preferences.
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
- The instrument picker only bundles BTC/USDT spot and perpetual contracts on Binance and OKX. Non-empty searches discover currently tradable spot and linear perpetual instruments through the optional `market.InstrumentSource` interface, with independent five-minute market-catalog caches. Search work is debounced and canceled/rejected on query changes, picker dismissal, and window closure; partial market failures keep successful results selectable. Recent successful explicit selections are deduplicated by full instrument identity, capped at 12, and atomically persisted to `trajon/instrument-history.json` under `os.UserConfigDir()`; clickable history appears below the search field.
- Keep network and other slow work off the UI thread. Reject stale market-load results when the selected instrument/timeframe changes, and close loaders when the window closes.

## MyGo UI rules

- Store application data, widget state, and `ui.Handle` values—not transient `ui.Element` or parts that expire after a build pass.
- Give stateful controls stable, unique keys before construction and preserve identity across rebuilds. Never derive keys from translated labels; the navigation helper avoids Onevra SidebarItem's label-based key.
- Sidebar navigation contains only Overview and Replay; instrument history remains in the instrument picker.
- Treat `marketToolbarInTitle` as fixed host-layout configuration: the native Shell enables it at startup, while standalone chart views leave it disabled. Never set it from `contentTitle` or another build callback; Shell constructs Content before ContentTitle, so a cross-frame marker can build both toolbar instances and bind one Handle twice on page entry. Test sidebar page transitions with content-before-title as well as title-before-content hosts.
- Localized UI text belongs in `internal/locales`, modeled after MyGo's updater catalogs. Startup resolves `mygo.App.Locale()` (BCP 47 or POSIX) to Simplified Chinese (`zh-Hans`) or English (`en`); unsupported languages/scripts, including Traditional Chinese, fall back to English. Settings provides System / 简体中文 / English with immediate switching and asynchronous atomic persistence to `trajon/locale.json` under `os.UserConfigDir()`. Missing/invalid preferences follow the system. Saving failures retain the active choice and offer retry. Language changes must preserve focused drafts, instrument identity/cache keys, annotations, replay state, and trades. Keep market errors structured and language-independent; localize them and source/market labels in presentation, preserving provider details. Apply Onevra component localization and update Shell labels on language changes.
- Use `Window.Update`/the established update path to publish background results; do not mutate UI state directly from worker goroutines.
- Keep drawing callbacks limited to painting and use existing MyGo/onevra components for controls, focus, keyboard behavior, and accessibility.
- Verify UI lifecycle and focus changes with the project’s existing tester patterns where applicable. Consult the pinned MyGo source before relying on an API.
- The chart toolbar's Indicators icon follows the drawing tools and a separator; there is no delete-annotation toolbar button (Delete/Backspace still remove the selected annotation). It opens a save/cancel dialog with a left-hand Volume / EMA / MACD list and right-hand settings. Volume and EMA20 remain visible by default; MACD starts hidden. EMA supports independently enabled, stably keyed lines with integer periods (1–10000), colors, and widths (0.5–4px); MACD uses configurable fast/slow/signal periods (12/26/9 by default, fast < slow), DIF/DEA colors, and positive/negative histogram colors. EMA is seeded by the first close; MACD's histogram is `2 * (DIF - DEA)`. Saved settings survive replay resets and committed instrument/timeframe changes within the application, but are not persisted to disk. Cancel, Esc, backdrop, and close discard drafts and restore opener focus. Indicator calculations run off the native UI thread, use immutable candle snapshots, reject stale/closed-window results, and recalculate after history prepends. Rendering and autoscaling use only enabled lines and revealed replay candles; MACD and volume use separate aligned panes.
- Replay-start selection previews all candles under a pointer-following gray future mask. Dragging only pans; an undragged chart click confirms the start, exits selection, and removes the selection marker. Confirmed replay still hides future candles until they are played.
- The floating action bar stays horizontally centered with a 52px bottom inset as the chart resizes or the sidebar toggles/resizes. Dragging preserves an offset from that center/bottom anchor; temporary boundary clamping must not cause drift after the chart expands again.
- The floating action bar shows replay-start selection before a start is confirmed. After confirmation its controls are grouped in order: play/pause and speed; order quantity, buy, and sell; reset and end, with thin separators between groups. Replay settings (initial capital defaults to 30000; leverage defaults to 1×) appear only alongside replay-start selection before a start is confirmed; they stay hidden after start confirmation, including while playing, paused, reset, or completed, and return alongside replay-start selection after discarding the End summary. Settings use a save/cancel dialog and persist across replay sessions and instrument/timeframe changes within the application.
- After replay-start confirmation, Space toggles playback, Up buys, and Down sells using the current order quantity while the chart or action-bar buttons have focus; the corresponding button tooltips display these shortcuts. While paused, Right reveals exactly one additional candle and remains paused, following the replay cursor and applying the same bracket exits, fill/error feedback, and completion handling as timed playback; it does not depend on order quantity and is unavailable while loading or playing. Keep quantity/annotation editing, speed selection, dialogs, and native toolbar/position-button activation independent of these commands. Shortcuts follow the same availability as their buttons, including completed replay and disabled trading.
- After replay-start confirmation, the floating action bar includes a playback-speed selector (0.5×, 1×, 2×, 5×, 10×; default 1×). At 1×, one candle is revealed per second. Changing speed while playing reschedules the next candle immediately without pausing or skipping candles. The chosen speed persists across pause/resume, reset, discarded sessions, and instrument/timeframe changes within the application. Completed replay disables speed changes.
- End first pauses playback and opens a simulation-trading summary without closing positions or clearing fills. The summary shows capital, equity, realized/unrealized P&L, return, closing-fill counts, win rate, and average win/loss ratio. Each fill that closes any quantity (including partial closes and reversals) counts as one outcome; breakeven closes count in the win-rate denominator, and unrealized P&L is excluded from win rate and win/loss ratio. Undefined metrics show a dash; wins without losses show an infinite ratio. Esc/backdrop dismisses the summary and retains the paused session. The bottom “放弃” action performs the original End logic: clear the replay session and restore all candles at the latest viewport position while preserving zoom and annotations. The “保存” action is visible but disabled until saving is implemented.
- After replay-start confirmation, the floating action bar also shows a compact order-quantity input and TrendingUp/TrendingDown buy/sell buttons. Quantity is in units of the instrument; step buttons and Up/Down adjust by 1, while manual entry still accepts fractional quantities. Trades fill at `candles[replayEnd-1].Close` and its timestamp, independently of hover/viewport, without pausing playback. A single net position supports longs and shorts; opposing orders close first and reverse any remainder. Initial capital plus realized/unrealized P&L and leverage limit opening exposure; closing remains possible under a margin deficit. No fees, slippage, or automatic liquidation are modeled. Completed replay disables trading. Reset, discarding the End summary, a new replay start, and a committed instrument/timeframe change clear positions and fills; saved replay settings remain intact.
- Non-flat replay positions show a horizontal average-entry line with a direction/quantity tag and a native close button; closing uses the latest revealed candle's Close for the entire position, not the order input. Hovering/dragging the line creates a take-profit or stop-loss preview (long: up=profit/down=loss; short: reversed); release commits and Esc cancels. Existing brackets can be dragged or canceled individually. Position, take-profit, and stop-loss tags stay compact and share a right-aligned edge. Bracket tags (including drag previews) show signed P&L amounts to two decimal places, not prices: `(exit price - average entry price) * signed remaining quantity`, excluding previously realized P&L and without multiplying by leverage. Price projection is frozen during a drag, and the chart range includes positions and active brackets. Brackets apply to the whole remaining position, survive same-direction additions/reductions, and clear on flat/reversal or session reset. Only candles after bracket creation/modification can trigger: High/Low touches fill at the line, gaps at Open, and same-candle dual touches prioritize stop-loss. Check exits before completing the final replay candle; successful exits do not pause playback. Completed replay leaves any unclosed position visible but disables trading and bracket edits.

## Change hygiene

- Keep user-facing copy concise and consistent across the Chinese/English catalogs; preserve existing Chinese UI terminology and do not add decorative or marketing copy.
- Do not commit generated artifacts, credentials, or local build output.
- Prefer a focused change over broad refactors. If behavior or public project conventions change, update this file and the relevant tests together.
