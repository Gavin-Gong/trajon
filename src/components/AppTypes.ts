export type AppView =
  | { view: "main" }
  | { view: "panel"; presetSymbol?: string; journalId?: string };
export type AppSection = "analysis" | "calendar" | "journal" | "settings";
export type AppTagVariant = "accent" | "danger" | "neutral" | "success" | "warning";
