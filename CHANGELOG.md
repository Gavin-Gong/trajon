# Changelog

## 0.0.5

### Added

- A trading journal with searchable history, an embedded editor, and a reusable floating quick-record panel.
- Manual plans, notes, signed fractional executions, trade reviews, tags, and validated local screenshots, with persistent drafts and atomic saves.
- A journal-performance Overview with whole-trade P&L, win rate, win/loss ratio, maximum drawdown, daily heatmaps, and instrument/tag breakdowns.

### Changed

- Use USD consistently for journal and replay capital and P&L, without currency selection or conversion.
- Refresh the application icon with the layered mountain mark.
- Update to MyGo 0.7.3 and Onevra 0.2026.16, using their built-in Markdown and localization support.

### Fixed

- Accept Longbridge OAuth grants with optional refresh tokens, omitted expiry, quoted expiry values, and long lifetimes; require reauthorization when an access-only grant expires.

## 0.0.4

### Added

- Longbridge browser authorization with encrypted local credential storage.
- US, Hong Kong, and Shanghai/Shenzhen stock and ETF search, candlestick data, and replay across all eleven timeframes.

### Changed

- Show the selected security's quote currency in replay settings and summaries.
- Improve the replay summary layout with an outcome-distribution chart and scrollable metrics.

## 0.0.3

### Added

- A timeframe dropdown with 1m, 3m, 5m, 15m, 30m, 1h, 2h, 4h, 1D, 1W, and 1M intervals on Binance and OKX.
- Starred timeframe shortcuts with local persistence across application restarts.

### Changed

- Show the current unstarred timeframe beside the dropdown arrow without adding it to saved favorites.
- Use a compact, left-aligned timeframe list with whole-row hover and selection backgrounds, color-only star highlights, and space for the scrollbar.
- Keep the timeframe dropdown and drawing tools reachable in narrow windows by scrolling only the starred shortcuts.

### Fixed

- Restore the last successfully loaded timeframe when rapid timeframe switching fails.
- Reset series-specific chart state after a successful timeframe change while preserving replay preferences and starred shortcuts.

## 0.0.2

### Added

- A Changelog dialog showing the app logo, name, current version, and a GitHub icon link.
- Background signed-update checks, Markdown release notes, installation progress, retries on failure, and relaunch after successful installation.

### Changed

- Keep the Changelog dialog compact when no update is available, and expand the scrollable release notes area for available updates while respecting small-window bounds.
- Migrate Simplified Chinese and English translations to MyGo JSON/ICU catalogs with typed message accessors.
- Synchronize app, native window, and shared component language changes, with system-language matching across ordered language preferences.
- Apply the Onevra theme and update MyGo to 0.5.1 and Onevra to 0.2026.9.

### Fixed

- Preserve stable chart and floating action bar control identity across UI rebuilds.
- Keep temporary development language overrides from overwriting saved language preferences.

## 0.0.1

### Added

- Binance and OKX spot and perpetual market data, instrument search, and recent selections.
- Candlestick charts with zooming, panning, trend lines, and text annotations.
- Volume, configurable EMA lines, and MACD indicators.
- Historical replay with start selection, play/pause, single-candle stepping, playback speeds, and keyboard shortcuts.
- Simulated long and short trading with initial capital, leverage, take-profit and stop-loss orders, and trading statistics.
- Simplified Chinese, English, and system language preferences.
