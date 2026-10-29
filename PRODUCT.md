# Trajon

<!-- impeccable:product-schema 1 -->

## Platform

Native desktop application. Current release targets are macOS on Apple Silicon and Intel; this is not a web or mobile surface.

## Product Purpose

A personal trading workspace combining manual trade journaling, journal-performance review, market charts, and historical replay practice.

## Positioning

Confirmed in the icon-design interview: present Trajon as a personal trading workspace, rather than a single-purpose chart, journal, or replay tool. Trajon abbreviates “trading journey”. The current logo brief focuses solely on successively higher mountains; the user explicitly dropped the earlier candlestick combination.

## Capabilities and Constraints

- Go and MyGo native application; preserve the pinned dependencies.
- Journals are manual records, and replay uses simulated orders; neither is live brokerage execution.
- Overview reports journal performance, not live-account or replay performance.
- Current application identity and release configuration live in `mygo.json`.

## Brand Commitments

- Keep the name Trajon, an abbreviation of “trading journey”.
- Current logo direction: mountains only, expressing “一山更比一山高”. Do not integrate candlesticks, candle-like cutouts, price axes, trading ticks, or other financial pictograms.
- Mountain crests become progressively higher from left to right. Continuous ranges and distinct mountains are both valid; avoid equal-height peaks, a lone symmetric summit, or an upward arrow standing in for mountains.
- Establish recognition through an authored mountain mark, without relying on T, TJ, or other letterforms.
- The current working mark is「层峦」: three sharp mountains, with the taller right mountain in front and overlapping those to its left. Reuse its existing three greens, ordered far light to near dark, and its rounded native icon tile.
- Keep the authoring sources in `resources/icon.svg` (complete application icon) and `resources/logo.svg` (transparent mountain mark). `resources/icon.png` remains the runtime/build PNG; change the shown icon only on user request.

## Evidence on Hand

- `AGENTS.md`: product capabilities and constraints.
- `resources/icon.svg`: complete application-icon source, including the native tile and current mountain placement.
- `resources/logo.svg`: transparent, standalone mountain-mark source.
- `resources/icon.png`: current 1024px application icon. MyGo packaging and the embedded application logo use the same PNG.
- `mygo.json`: application identity and release configuration.

## Open Decisions

-「层峦」is the current working app icon. At the user's request, only its core SVGs are retained and the exploration directory has been removed. This does not imply a release or version change.
- Primary audience beyond the personal-workspace positioning is not yet specified.
