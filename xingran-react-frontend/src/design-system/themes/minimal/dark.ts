/**
 * 极简现代风主题 - 深色模式
 * 特点：深邃背景、高对比度、干净线条
 * WCAG AA 对比度：文字 #EDEDED 在 #0A0A0A 上 ~18:1 ✓
 */

import type { ThemeConfig } from "@/types/theme";
import { spacing } from "../../tokens/spacing";
import { fontFamily } from "../../tokens/typography";

export const minimalDark: ThemeConfig = {
  id: "minimal",
  name: "极简现代 (Dark)",
  description: "深邃背景、干净线条、高对比度，专注夜间使用",

  colors: {
    primary: ["#EDEDED", "#D4D4D4", "#A3A3A3", "#737373", "#525252"],
    secondary: ["#171717", "#262626", "#404040", "#525252", "#737373"],
    accent: ["#EDEDED", "#D4D4D4", "#A3A3A3"],
    neutral: [
      "#FAFAFA",
      "#EDEDED",
      "#D4D4D4",
      "#A3A3A3",
      "#737373",
      "#525252",
      "#404040",
      "#262626",
      "#0A0A0A",
    ],

    success: ["#4ADE80", "#22C55E", "#16A34A"],
    warning: ["#FBBF24", "#F59E0B", "#D97706"],
    error: ["#F87171", "#EF4444", "#DC2626"],
    info: ["#60A5FA", "#3B82F6", "#2563EB"],
    processing: ["#34D399", "#10B981", "#059669"],

    background: {
      primary: "#0A0A0A",
      secondary: "#171717",
      tertiary: "#262626",
      surface: "#171717",
      elevated: "#262626",
    },

    text: {
      primary: "#EDEDED",
      secondary: "#A3A3A3",
      tertiary: "#737373",
      disabled: "#525252",
      inverse: "#0A0A0A",
    },

    border: {
      primary: "#27272A",
      secondary: "#3F3F46",
      divider: "#27272A",
    },
  },

  spacing: {
    xs: spacing.xs,
    sm: spacing.sm,
    md: spacing.lg,
    lg: spacing.xl,
    xl: spacing["2xl"],
    "2xl": spacing["3xl"],
    "3xl": spacing["4xl"],
  },

  typography: {
    fontFamily: fontFamily.sans,
    fontSize: {
      xs: "12px",
      sm: "14px",
      base: "16px",
      lg: "18px",
      xl: "20px",
      "2xl": "24px",
      "3xl": "30px",
    },
    fontWeight: {
      normal: "400",
      medium: "500",
      semibold: "600",
      bold: "700",
    },
    lineHeight: {
      tight: "1.25",
      normal: "1.5",
      relaxed: "1.625",
    },
  },

  shadows: {
    xs: "none",
    sm: "0 1px 2px rgba(0, 0, 0, 0.3)",
    md: "0 2px 4px rgba(0, 0, 0, 0.4)",
    lg: "0 4px 8px rgba(0, 0, 0, 0.5)",
    xl: "0 8px 16px rgba(0, 0, 0, 0.6)",
    "2xl": "0 12px 24px rgba(0, 0, 0, 0.7)",
    inner: "inset 0 1px 2px rgba(0, 0, 0, 0.5)",
  },

  radius: {
    sm: "2px",
    md: "4px",
    lg: "6px",
    xl: "8px",
    "2xl": "12px",
    full: "9999px",
  },

  effects: {
    minimal: {
      borderWidth: "1px",
      borderColor: "#27272A",
    },
    transition: {
      fast: "150ms cubic-bezier(0.4, 0, 0.2, 1)",
      base: "200ms cubic-bezier(0.4, 0, 0.2, 1)",
      slow: "300ms cubic-bezier(0.4, 0, 0.2, 1)",
    },
  },
};
