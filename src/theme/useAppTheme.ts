import { useMemo } from "react";
import type { ConfigProviderProps } from "antd";
import { theme } from "antd";
import { createStyles } from "antd-style";
import clsx from "clsx";

const fontFamily = '"SF Pro Text", "SF Pro Display", "Helvetica Neue", "Segoe UI", sans-serif';

type AppThemeStyles = {
  buttonPrimary: string;
  buttonDefault: string;
  buttonDanger: string;
  inputRoot: string;
  inputElement: string;
  inputError: string;
  selectRoot: string;
  selectPopup: string;
};

const useStyles = createStyles(({ css, prefixCls }) => ({
  buttonPrimary: css`
    &.${prefixCls}-btn {
      background-color: #18181b;
      border: 1px solid #18181b;
      color: #ffffff;
      font-weight: 500;
      transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);

      &:hover,
      &:focus-visible {
        background-color: #27272a;
        border-color: #27272a;
        color: #ffffff;
      }

      &:active {
        background-color: #09090b;
        border-color: #09090b;
      }
    }
  `,
  buttonDefault: css`
    &.${prefixCls}-btn {
      background-color: #ffffff;
      border: 1px solid #e4e4e7;
      color: #18181b;
      font-weight: 500;
      transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);

      &:hover,
      &:focus-visible {
        background-color: #f4f4f5;
        border-color: #d4d4d8;
        color: #18181b;
      }

      &:active {
        background-color: #e4e4e7;
        border-color: #d4d4d8;
      }
    }
  `,
  buttonDanger: css`
    &.${prefixCls}-btn {
      background-color: #dc2626;
      border: 1px solid #dc2626;
      color: #ffffff;
      font-weight: 500;

      &:hover,
      &:focus-visible {
        background-color: #b91c1c;
        border-color: #b91c1c;
        color: #ffffff;
      }
    }
  `,
  inputRoot: css`
    &.${prefixCls}-input {
      border-color: #e4e4e7;
      color: #18181b;
      transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);

      &:hover {
        border-color: #a1a1aa;
      }

      &:focus,
      &:focus-within {
        border-color: #18181b;
        box-shadow: none;
      }
    }
  `,
  inputElement: css`
    color: #18181b;
  `,
  inputError: css`
    &.${prefixCls}-input {
      border-color: #dc2626;

      &:hover,
      &:focus,
      &:focus-within {
        border-color: #dc2626;
      }
    }
  `,
  selectRoot: css`
    &.${prefixCls}-select {
      .${prefixCls}-select-selector {
        border-color: #e4e4e7;
        border-radius: 6px;
        transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
      }

      &:hover .${prefixCls}-select-selector {
        border-color: #a1a1aa;
      }

      &.${prefixCls}-select-focused .${prefixCls}-select-selector {
        border-color: #18181b;
        box-shadow: none;
      }
    }
  `,
  selectPopup: css`
    border-radius: 8px;
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -2px rgba(0, 0, 0, 0.1);
  `,
}));

export function createAppThemeConfig(styles: AppThemeStyles): ConfigProviderProps {
  return {
      theme: {
        algorithm: theme.defaultAlgorithm,
        token: {
          colorPrimary: "#262626",
          colorSuccess: "#22c55e",
          colorWarning: "#f97316",
          colorError: "#ef4444",
          colorInfo: "#262626",
          colorTextBase: "#262626",
          colorBgBase: "#ffffff",
          colorPrimaryBg: "#f5f5f5",
          colorPrimaryBgHover: "#e5e5e5",
          colorPrimaryBorder: "#d4d4d4",
          colorPrimaryBorderHover: "#a3a3a3",
          colorPrimaryHover: "#404040",
          colorPrimaryActive: "#171717",
          colorPrimaryText: "#262626",
          colorPrimaryTextHover: "#404040",
          colorPrimaryTextActive: "#171717",
          colorSuccessBg: "#f0fdf4",
          colorSuccessBgHover: "#dcfce7",
          colorSuccessBorder: "#bbf7d0",
          colorSuccessBorderHover: "#86efac",
          colorSuccessHover: "#16a34a",
          colorSuccessActive: "#15803d",
          colorSuccessText: "#16a34a",
          colorSuccessTextHover: "#16a34a",
          colorSuccessTextActive: "#15803d",
          colorWarningBg: "#fff7ed",
          colorWarningBgHover: "#fed7aa",
          colorWarningBorder: "#fdba74",
          colorWarningBorderHover: "#fb923c",
          colorWarningHover: "#ea580c",
          colorWarningActive: "#c2410c",
          colorWarningText: "#ea580c",
          colorWarningTextHover: "#ea580c",
          colorWarningTextActive: "#c2410c",
          colorErrorBg: "#fef2f2",
          colorErrorBgHover: "#fecaca",
          colorErrorBorder: "#fca5a5",
          colorErrorBorderHover: "#f87171",
          colorErrorHover: "#dc2626",
          colorErrorActive: "#b91c1c",
          colorErrorText: "#dc2626",
          colorErrorTextHover: "#dc2626",
          colorErrorTextActive: "#b91c1c",
          colorInfoBg: "#f5f5f5",
          colorInfoBgHover: "#e5e5e5",
          colorInfoBorder: "#d4d4d4",
          colorInfoBorderHover: "#a3a3a3",
          colorInfoHover: "#404040",
          colorInfoActive: "#171717",
          colorInfoText: "#262626",
          colorInfoTextHover: "#404040",
          colorInfoTextActive: "#171717",
          colorText: "#262626",
          colorTextSecondary: "#525252",
          colorTextTertiary: "#737373",
          colorTextQuaternary: "#a3a3a3",
          colorTextDisabled: "#a3a3a3",
          colorBgContainer: "#ffffff",
          colorBgElevated: "#ffffff",
          colorBgLayout: "#fafafa",
          colorBgSpotlight: "rgba(38, 38, 38, 0.85)",
          colorBgMask: "rgba(38, 38, 38, 0.45)",
          colorBorder: "#e5e5e5",
          colorBorderSecondary: "#f5f5f5",
          borderRadius: 10,
          borderRadiusXS: 2,
          borderRadiusSM: 6,
          borderRadiusLG: 14,
          padding: 16,
          paddingSM: 12,
          paddingLG: 24,
          margin: 16,
          marginSM: 12,
          marginLG: 24,
          boxShadow: "0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px -1px rgba(0, 0, 0, 0.1)",
          boxShadowSecondary:
            "0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -2px rgba(0, 0, 0, 0.1)",
          fontFamily,
        },
        components: {
          Button: {
            primaryShadow: "none",
            defaultShadow: "none",
            dangerShadow: "none",
            defaultBorderColor: "#e4e4e7",
            defaultColor: "#18181b",
            defaultBg: "#ffffff",
            defaultHoverBg: "#f4f4f5",
            defaultHoverBorderColor: "#d4d4d8",
            defaultHoverColor: "#18181b",
            defaultActiveBg: "#e4e4e7",
            defaultActiveBorderColor: "#d4d4d8",
            borderRadius: 6,
          },
          Input: {
            activeShadow: "none",
            hoverBorderColor: "#a1a1aa",
            activeBorderColor: "#18181b",
            borderRadius: 6,
          },
          Select: {
            optionSelectedBg: "#f4f4f5",
            optionActiveBg: "#fafafa",
            optionSelectedFontWeight: 500,
            borderRadius: 6,
          },
          Alert: {
            borderRadiusLG: 8,
          },
          Modal: {
            borderRadiusLG: 12,
          },
          Progress: {
            defaultColor: "#18181b",
            remainingColor: "#f4f4f5",
          },
          Steps: {
            iconSize: 32,
          },
          Switch: {
            trackHeight: 24,
            trackMinWidth: 44,
            innerMinMargin: 4,
            innerMaxMargin: 24,
          },
          Checkbox: {
            borderRadiusSM: 4,
          },
          Slider: {
            trackBg: "#f4f4f5",
            trackHoverBg: "#e4e4e7",
            handleSize: 18,
            handleSizeHover: 20,
            railSize: 6,
          },
          ColorPicker: {
            borderRadius: 6,
          },
        },
      },
      button: {
        classNames: ({ props }) => ({
          root: clsx(
            props.danger && styles.buttonDanger,
            props.type === "primary" && styles.buttonPrimary,
            (!props.type || props.type === "default") && styles.buttonDefault,
          ),
        }),
      },
      input: {
        classNames: ({ props }) => ({
          root: clsx(styles.inputRoot, props.status === "error" && styles.inputError),
          input: styles.inputElement,
        }),
      },
      select: {
        classNames: {
          root: styles.selectRoot,
          popup: {
            root: styles.selectPopup,
          },
        },
      },
    };
}

export function useAppTheme(): ConfigProviderProps {
  const { styles } = useStyles();

  return useMemo(() => createAppThemeConfig(styles), [styles]);
}

export { fontFamily };
