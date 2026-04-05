# Tailwind-to-Ant Design Migration Design

**Date:** 2026-04-05

## Context

The current desktop app is a mixed UI stack:

- Ant Design is already used for primitives such as `Layout`, `Menu`, `Button`, `Calendar`, `Modal`, `Form`, `Input`, `Select`, `Switch`, `Tag`, `Badge`, `FloatButton`, `Row`, and `Col`
- Tailwind is still the primary layout and presentation layer inside `src/tradingPanelApp.tsx`
- Global visual identity is expressed through `src/App.css` with Tailwind import, CSS custom properties, and `.ant-*` overrides

The user explicitly chose to migrate toward a **standard Ant Design visual style**, not preserve the current Tailwind-driven visual identity.

## Goal

Remove Tailwind from the runtime and codebase UI layer, migrate the app to a more standard Ant Design structure and styling model, and keep the existing functionality intact.

## Scope

### In scope

- Add an Ant Design root theme/provider layer
- Replace Tailwind utility-based layout and presentation in `src/tradingPanelApp.tsx`
- Prefer standard Ant Design building blocks such as `Layout`, `Card`, `Flex`, `Space`, `Typography`, `List`, `Descriptions`, `Form`, `Input`, `Segmented`, and `Tabs` where they fit the current workflows
- Reduce or remove ad-hoc `.ant-*` CSS overrides when token/theme configuration can replace them
- Remove Tailwind imports, Vite integration, and dependencies after migration
- Keep current product behavior, data flow, and journal interactions working

### Out of scope

- Large-scale feature additions
- Broad domain-model refactors unrelated to UI migration
- Major component tree decomposition before migration
- Pixel-perfect preservation of the current Tailwind design language

## Design Decisions

### 1. Phased migration instead of one-shot rewrite

The migration will happen in phases so the app remains verifiable during the transition:

1. Add regression coverage around the current desktop shell and core flows
2. Introduce root-level Ant Design theme configuration
3. Convert the main desktop shell and analytics/calendar/settings sections toward Ant Design composition
4. Convert the panel surface and form-heavy areas
5. Remove Tailwind integration and dead styling

This keeps the blast radius manageable and gives clear checkpoints for tests and build validation.

### 2. Migrate in place before splitting files

`src/tradingPanelApp.tsx` is large, but splitting it while also replacing the styling system would introduce unnecessary structural risk. The primary migration will happen in place. Small extractions are allowed only when they reduce repetition without expanding scope.

### 3. Standard Ant Design look over custom token fidelity

The target is not a visual clone of the current interface. Existing CSS custom properties such as `--app-accent`, `--app-surface`, and `--app-shadow-soft` should be reduced or removed where Ant Design theme tokens and component defaults can replace them. The final app should look recognizably closer to Ant Design than the current custom Tailwind skin.

### 4. Use Ant Design composition for layout and spacing

Tailwind utility patterns like `grid`, `flex`, `gap-*`, `space-y-*`, `rounded-*`, `border-*`, and `bg-*` should be replaced with Ant Design composition patterns first:

- `Layout` for page shell
- `Flex` / `Space` for spacing and alignment
- `Card` / `List` / `Descriptions` / `Statistic` for structured content blocks
- `Typography` for headings and body text
- `Form` inputs for standard interactive controls where practical

Only keep custom CSS for app-wide document setup or genuinely unsupported layout cases.

### 5. Root theme via `ConfigProvider`

Ant Design should be configured at the app entry so the migration moves styling responsibility away from raw CSS overrides and toward theme tokens. This creates a cleaner long-term base than continuing to style internal `.ant-*` selectors globally.

## Proposed Architecture

### App entry

- `src/main.tsx` and/or `src/App.tsx` will wrap the app in `ConfigProvider`
- Theme values will centralize base choices such as primary color, border radius, and typography scale

### Global CSS

- `src/App.css` will stop importing Tailwind
- Global CSS will shrink to document-level concerns and any minimal app-scoped rules still needed after the Ant Design migration
- Existing `.ant-*` overrides will be removed or reduced significantly

### Main UI module

`src/tradingPanelApp.tsx` remains the main migration surface. The key target areas are:

- `MainSurface`: sidebar/shell normalization
- `AnalyticsPanel`: card-heavy dashboard content
- `CalendarHistorySection`: calendar and day-history display
- `SettingsSection`: settings cards and controls
- `PanelSurface`: floating panel UI and quick capture flow
- supporting reusable visual blocks such as metric/status sections

## Risks and Mitigations

### Risk: visual regressions

There are no visual snapshot tests. Mitigation:

- add regression tests for important text/controls before refactoring
- keep migration phased
- verify build and behavior after each logical milestone

### Risk: large diff in `tradingPanelApp.tsx`

Mitigation:

- migrate section by section
- keep behavior unchanged
- avoid mixing unrelated logic refactors into the UI migration

### Risk: hybrid styling during transition

Mitigation:

- establish `ConfigProvider` early
- convert top-level layout first
- remove Tailwind only after all live usage is gone

## Testing Strategy

- Extend existing Vitest + Testing Library coverage before major refactors
- Add assertions for the app shell and key actions/sections that should remain accessible after migration
- Run targeted tests during each phase
- Run full test suite and production build before declaring completion

## Success Criteria

The migration is successful when all of the following are true:

- `src/tradingPanelApp.tsx` no longer relies on Tailwind utility classes for UI styling
- `src/App.css` no longer imports Tailwind
- `vite.config.ts` and `package.json` no longer include Tailwind integration/dependencies
- the application is wrapped in a root Ant Design theme/provider
- tests and build pass
- the resulting interface follows a more standard Ant Design visual language than the current custom Tailwind presentation
