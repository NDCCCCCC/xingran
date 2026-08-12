/**
 * 扁平化2.0主题 - 浅色模式
 * 特点：扁平设计+微妙渐变、鲜艳配色、清晰图标、活力十足
 */

import type { ThemeConfig } from "@/types/theme";
import { spacing } from "../../tokens/spacing";
import { fontFamily } from "../../tokens/typography";
import { radius } from "../../tokens/shadows";

export const flat2Light: ThemeConfig = {
  id: "flat2.0",
  name: "扁平化2.0",
  description: "扁平设计配微妙渐变和阴影，鲜艳配色充满活力",

  colors: {
    // 主色：鲜艳蓝色
    primary: ["#93c5fd", "#60a5fa", "#3b82f6", "#2563eb", "#1d4ed8"],
    secondary: ["#c4b5fd", "#a78bfa", "#8b5cf6", "#7c3aed", "#6d28d9"],
    accent: ["#f472b6", "#ec4899", "#db2777"],
    neutral: ["#f8fafc", "#f1f5f9", "#e2e8f0", "#cbd5e1", "#94a3b8", "#64748b", "#475569", "#334155", "#1e293b"],

    // 功能色：鲜艳
    success: ["#4ade80", "#22c55e", "#16a34a"],
    warning: ["#fbbf24", "#f59e0b", "#d97706"],
    error: ["#f87171", "#ef4444", "#dc2626"],
    info: ["#60a5fa", "#3b82f6", "#2563eb"],
    processing: ["#4ade80", "#22c55e", "#16a34a"],

    // 背景
    background: {
      primary: "#ffffff",
      secondary: "#f8fafc",
      tertiary: "#f1f5f9",
      surface: "#ffffff",
      elevated: "#ffffff",
    },

    // 文字
    text: {
      primary: "#0f172a",
      secondary: "#475569",
      tertiary: "#64748b",
      disabled: "#94a3b8",
      inverse: "#ffffff",
    },

    // 边框
    border: {
      primary: "#e2e8f0",
      secondary: "#cbd5e1",
      divider: "#f1f5f9",
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
    xs: "0 1px 2px rgba(59, 130, 246, 0.1)",
    sm: "0 2px 4px rgba(59, 130, 246, 0.15)",
    md: "0 4px 8px rgba(59, 130, 246, 0.2)",
    lg: "0 8px 16px rgba(59, 130, 246, 0.25)",
    xl: "0 12px 24px rgba(59, 130, 246, 0.3)",
    "2xl": "0 20px 40px rgba(59, 130, 246, 0.35)",
    inner: "inset 0 2px 4px rgba(59, 130, 246, 0.1)",
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
    flat2: {
      gradient: "linear-gradient(135deg, #667eea 0%, #764ba2 100%)",
      hoverLift: "translateY(-2px)",
    },
    transition: {
      fast: "150ms cubic-bezier(0.4, 0, 0.2, 1)",
      base: "250ms cubic-bezier(0.4, 0, 0.2, 1)",
      slow: "400ms cubic-bezier(0.4, 0, 0.2, 1)",
    },
  },
};
