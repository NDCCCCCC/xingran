/**
 * 墨绿琥珀主题 - 深色模式
 * 特点：墨绿炭黑背景 + 提亮绿主色 + 米金文字（源自登录页 v4 品牌配色）
 * WCAG AA 对比度：文字 #fef3c7 在 #0f1512 上 ~14:1 ✓
 */

import type { ThemeConfig } from "@/types/theme";
import { spacing } from "../../tokens/spacing";
import { fontFamily } from "../../tokens/typography";
import { radius } from "../../tokens/shadows";

export const inkAmberDark: ThemeConfig = {
  id: "ink-amber",
  name: "墨绿琥珀 (Dark)",
  description: "墨绿炭黑背景配提亮绿与米金文字，源自登录页品牌配色",

  // AntD 组件主色：琥珀金（暗底提亮，登录页暗面板同款点缀，= accent[1]）
  antdPrimary: "#d4a574",

  colors: {
    // 主色：暗底下提亮绿保证对比度（索引 2 = 主题主色 #22c55e）
    primary: ["#052e16", "#14532d", "#22c55e", "#4ade80", "#86efac"],
    secondary: ["#fafaf9", "#f5f5f4", "#e7e5e4", "#a8a29e", "#78716c"],
    accent: ["#8a6534", "#d4a574", "#f0c896"],
    neutral: [
      "#fafaf9",
      "#f5f5f4",
      "#e7e5e4",
      "#d6d3d1",
      "#a8a29e",
      "#78716c",
      "#57534e",
      "#44403c",
      "#1c1917",
    ],

    success: ["#73d13d", "#52c41a", "#389e0d"],
    warning: ["#ffd666", "#faad14", "#d48806"],
    error: ["#ff7875", "#ff4d4f", "#cf1322"],
    info: ["#69c0ff", "#1890ff", "#096dd9"],
    processing: ["#52c41a", "#389e0d", "#237804"],

    // 背景：墨绿炭黑系列
    background: {
      primary: "#0f1512", // 墨绿炭底
      secondary: "#141b17", // 次级背景
      tertiary: "#1a231e", // 三级背景
      surface: "#121a15", // 表面背景
      elevated: "#182019", // 悬浮背景
    },

    // 文字：米金主文字
    text: {
      primary: "#fef3c7", // 主要文字 - 米金
      secondary: "#e7e5e4", // 次要文字 - 浅石灰
      tertiary: "#a8a29e", // 三级文字 - 石灰
      disabled: "#57534e", // 禁用文字
      inverse: "#1c1917", // 反色文字 - 极深墨
    },

    // 边框：琥珀半透明（暗底适配）
    border: {
      primary: "rgba(212, 165, 116, 0.25)", // 琥珀半透明
      secondary: "rgba(212, 165, 116, 0.12)", // 浅琥珀
      divider: "rgba(254, 243, 199, 0.08)", // 分割线
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
    xs: "0 1px 2px rgba(0, 0, 0, 0.3)",
    sm: "0 2px 4px rgba(0, 0, 0, 0.4)",
    md: "0 4px 8px rgba(0, 0, 0, 0.5)",
    lg: "0 8px 16px rgba(0, 0, 0, 0.6)",
    xl: "0 16px 32px rgba(0, 0, 0, 0.7)",
    "2xl": "0 24px 48px rgba(0, 0, 0, 0.8)",
    inner: "inset 0 1px 3px rgba(0, 0, 0, 0.3)",
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
    transition: {
      fast: "150ms cubic-bezier(0.4, 0, 0.2, 1)",
      base: "250ms cubic-bezier(0.4, 0, 0.2, 1)",
      slow: "350ms cubic-bezier(0.4, 0, 0.2, 1)",
    },
  },
};
