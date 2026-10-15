# Changelog

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
