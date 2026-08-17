/**
 * 墨绿琥珀主题 - 浅色模式
 * 特点：米白背景 + 墨绿主色 + 琥珀金点缀（源自登录页 v4 品牌配色）
 */

import type { ThemeConfig } from "@/types/theme";
import { spacing } from "../../tokens/spacing";
import { fontFamily } from "../../tokens/typography";
import { radius } from "../../tokens/shadows";

export const inkAmberLight: ThemeConfig = {
  id: "ink-amber",
  name: "墨绿琥珀",
  description: "米白背景配墨绿主色与琥珀金点缀，源自登录页品牌配色",

  colors: {
    // 主色：墨绿系列（索引 2 = 主题主色 #166534）
    primary: [
      "#f0fdf4", // 50 - 浅绿白
      "#dcfce7", // 100 - 浅绿
      "#166534", // 200 - 墨绿（主题色）
      "#14532d", // 300 - 深墨绿
      "#0f3d22", // 400 - 极深墨绿
    ],
    secondary: ["#fafaf9", "#f5f5f4", "#e7e5e4", "#a8a29e", "#78716c"],
    accent: ["#d4a574", "#b8854c", "#8a6534"],
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

    // 功能色 - 柔和版本
    success: ["#73d13d", "#52c41a", "#389e0d"],
    warning: ["#ffd666", "#faad14", "#d48806"],
    error: ["#ff7875", "#ff4d4f", "#cf1322"],
    info: ["#69c0ff", "#1890ff", "#096dd9"],
    processing: ["#52c41a", "#389e0d", "#237804"],

    // 背景：米白暖色系列（登录页同款）
    background: {
      primary: "#faf7f2", // 米白
      secondary: "#f3efe8", // 次级米白
      tertiary: "#f0ebe0", // 三级米白
      surface: "#ffffff", // 表面背景
      elevated: "#ffffff", // 悬浮背景
    },

    // 文字：stone 深色系列（登录页同款）
    text: {
      primary: "#1c1917", // 主要文字 - 极深墨
      secondary: "#44403c", // 次要文字 - 深石灰
      tertiary: "#78716c", // 三级文字 - 中石灰
      disabled: "#a8a29e", // 禁用文字
      inverse: "#fef3c7", // 反色文字 - 米金（登录品牌面板同款）
    },

    // 边框：琥珀半透明（登录页眉线/badge 同款）
    border: {
      primary: "rgba(212, 165, 116, 0.3)", // 琥珀半透明
      secondary: "rgba(212, 165, 116, 0.15)", // 浅琥珀
      divider: "rgba(28, 25, 23, 0.06)", // 分割线
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
    xs: "0 1px 2px rgba(0, 0, 0, 0.05)",
    sm: "0 2px 4px rgba(0, 0, 0, 0.06)",
    md: "0 4px 8px rgba(0, 0, 0, 0.08)",
    lg: "0 8px 16px rgba(0, 0, 0, 0.1)",
    xl: "0 16px 32px rgba(0, 0, 0, 0.12)",
    "2xl": "0 24px 48px rgba(0, 0, 0, 0.15)",
    inner: "inset 0 1px 3px rgba(0, 0, 0, 0.05)",
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
