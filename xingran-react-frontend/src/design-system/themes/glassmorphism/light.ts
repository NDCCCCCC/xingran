/**
 * 玻璃拟态主题 - 浅色模式
 * 特点：半透明背景、背景模糊、柔和光效、现代感
 */

import type { ThemeConfig } from "@/types/theme";
import { spacing } from "../../tokens/spacing";
import { fontFamily } from "../../tokens/typography";
import { radius } from "../../tokens/shadows";

export const glassmorphismLight: ThemeConfig = {
  id: "glassmorphism",
  name: "玻璃拟态",
  description: "半透明背景、背景模糊效果，营造轻盈现代感",

  colors: {
    // 主色：紫色系
    primary: ["#a5b4fc", "#818cf8", "#6366f1", "#4f46e5", "#4338ca"],
    secondary: ["#f3e8ff", "#e9d5ff", "#d8b4fe", "#c084fc", "#a855f7"],
    accent: ["#c084fc", "#a855f7", "#9333ea"],
    neutral: ["#faf5ff", "#f3e8ff", "#e9d5ff", "#d8b4fe", "#a78bfa", "#8b5cf6", "#7c3aed", "#6d28d9", "#5b21b6"],

    // 功能色（柔和版）
    success: ["#86efac", "#22c55e", "#16a34a"],
    warning: ["#fcd34d", "#f59e0b", "#d97706"],
    error: ["#fca5a5", "#ef4444", "#dc2626"],
    info: ["#93c5fd", "#3b82f6", "#2563eb"],
    processing: ["#86efac", "#22c55e", "#16a34a"],

    // 背景：半透明
    background: {
      primary: "rgba(255, 255, 255, 0.75)",
      secondary: "rgba(255, 255, 255, 0.5)",
      tertiary: "rgba(255, 255, 255, 0.3)",
      surface: "rgba(255, 255, 255, 0.85)",
      elevated: "rgba(255, 255, 255, 0.9)",
    },

    // 文字
    text: {
      primary: "#1e1b4b",
      secondary: "#4c1d95",
      tertiary: "#6d28d9",
      disabled: "#8b5cf6",
      inverse: "#ffffff",
    },

    // 边框：半透明
    border: {
      primary: "rgba(255, 255, 255, 0.4)",
      secondary: "rgba(255, 255, 255, 0.2)",
      divider: "rgba(255, 255, 255, 0.15)",
    },
  },

  spacing: {
    xs: spacing.xs,
    sm: spacing.sm,
    md: spacing.md,
    lg: spacing.lg,
    xl: spacing.xl,
    "2xl": spacing["2xl"],
    "3xl": spacing["3xl"],
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
    xs: "0 2px 8px rgba(99, 102, 241, 0.08)",
    sm: "0 4px 12px rgba(99, 102, 241, 0.12)",
    md: "0 8px 20px rgba(99, 102, 241, 0.15)",
    lg: "0 12px 32px rgba(99, 102, 241, 0.18)",
    xl: "0 20px 48px rgba(99, 102, 241, 0.2)",
    "2xl": "0 32px 64px rgba(99, 102, 241, 0.25)",
    inner: "inset 0 2px 8px rgba(99, 102, 241, 0.1)",
  },

  radius: {
    sm: radius.sm,
    md: radius.md,
    lg: radius.lg,
    xl: radius.xl,
    "2xl": radius["2xl"],
    full: radius.full,
  },

  effects: {
    glass: {
      blur: "20px",
      opacity: "0.75",
      border: "1px solid rgba(255, 255, 255, 0.3)",
      saturation: "180%",
    },
    transition: {
      fast: "200ms cubic-bezier(0.4, 0, 0.2, 1)",
      base: "300ms cubic-bezier(0.4, 0, 0.2, 1)",
      slow: "500ms cubic-bezier(0.4, 0, 0.2, 1)",
    },
  },
};
