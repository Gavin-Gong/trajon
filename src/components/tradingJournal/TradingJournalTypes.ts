export type AppView =
  | { view: "main" }
  | { view: "panel"; presetSymbol?: string; journalId?: string };
export type AppSection = "dashboard" | "calendar" | "settings";
export type AppTagVariant = "accent" | "danger" | "neutral" | "success" | "warning";
