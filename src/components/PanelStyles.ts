import type { CSSProperties } from "react";

export const shellCardStyle: CSSProperties = {};

export const mutedShellCardStyle: CSSProperties = {
  ...shellCardStyle,
  background: "rgba(15, 23, 42, 0.04)",
  boxShadow: "none",
};

export const dashedShellCardStyle: CSSProperties = {
  ...mutedShellCardStyle,
  background: "transparent",
};

export const panelCardStyle: CSSProperties = {
  ...shellCardStyle,
};

export const plainPanelSectionStyle: CSSProperties = {
  padding: 16,
};

export const plainMutedBlockStyle: CSSProperties = {
  padding: 12,
};

export const plainDashedBlockStyle: CSSProperties = {
  ...plainMutedBlockStyle,
  background: "transparent",
  border: "1px dashed rgba(15, 23, 42, 0.14)",
};
