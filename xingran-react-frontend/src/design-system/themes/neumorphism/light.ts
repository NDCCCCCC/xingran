/**
 * 新拟态主题 - 浅色模式
 * 特点：柔和阴影浮雕效果、圆润设计、同色调高低光
 */

import type { ThemeConfig } from "@/types/theme";
import { spacing } from "../../tokens/spacing";
import { fontFamily } from "../../tokens/typography";

// 新拟态专用背景色
const NEU_BG = "#e0e5ec";
const NEU_LIGHT = "rgba(255, 255, 255, 0.85)";
const NEU_DARK = "rgba(163, 177, 198, 0.65)";

export const neumorphismLight: ThemeConfig = {
  id: "neumorphism",
  name: "新拟态",
  description: "柔和的阴影浮雕效果，营造质感和深度",

  colors: {
    // 主色：靛青色系（配合背景）
    primary: ["#e0e7ff", "#c7d2fe", "#a5b4fc", "#818cf8", "#6366f1"],
    secondary: ["#f5f3ff", "#ede9fe", "#ddd6fe", "#c4b5fd", "#a855f7"],
    accent: ["#818cf8", "#6366f1", "#4f46e5"],
    neutral: [
      "#e0e5ec",
      "#d1d9e6",
      "#c3c9d2",
      "#94a3b8",
      "#64748b",
      "#475569",
      "#334155",
      "#1e293b",
      "#0f172a",
    ],

    // 功能色（柔和版）
    success: ["#bbf7d0", "#4ade80", "#22c55e"],
    warning: ["#fde047", "#fbbf24", "#f59e0b"],
    error: ["#fecaca", "#f87171", "#ef4444"],
    info: ["#bfdbfe", "#60a5fa", "#3b82f6"],
    processing: ["#bbf7d0", "#4ade80", "#22c55e"],

    // 背景：新拟态专用
    background: {
      primary: NEU_BG,
      secondary: "#d1d9e6",
      tertiary: "#cbd5e1",
      surface: NEU_BG,
      elevated: "#e8edf3",
    },

    // 文字
    text: {
      primary: "#4a5568",
      secondary: "#5a6578",
      tertiary: "#6a7588",
      disabled: "#9ca3af",
      inverse: "#ffffff",
    },

    // 边框：与背景同色
    border: {
      primary: NEU_BG,
      secondary: "#d1d9e6",
      divider: "transparent",
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
    xs: "none",
    sm: "none",
    md: "none",
    lg: "none",
    xl: "none",
    "2xl": "none",
    inner: "none",
  },

  radius: {
    sm: "12px", // 圆润
    md: "16px", // 圆润
    lg: "20px", // 圆润
    xl: "24px", // 圆润
    "2xl": "32px",
    full: "9999px",
  },

  effects: {
    neumorphic: {
      light: NEU_LIGHT,
      dark: NEU_DARK,
      radius: "16px",
      distance: "8px",
    },
    transition: {
      fast: "250ms cubic-bezier(0.4, 0, 0.2, 1)",
      base: "350ms cubic-bezier(0.4, 0, 0.2, 1)",
      slow: "500ms cubic-bezier(0.4, 0, 0.2, 1)",
    },
  },
};
