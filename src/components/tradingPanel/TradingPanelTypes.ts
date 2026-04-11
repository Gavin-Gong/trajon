export type AppView =
  | { view: "main" }
  | { view: "panel"; presetSymbol?: string };
export type AppSection = "analytics" | "chart" | "calendar" | "settings";
export type AppTagVariant = "accent" | "danger" | "neutral" | "success" | "warning";
