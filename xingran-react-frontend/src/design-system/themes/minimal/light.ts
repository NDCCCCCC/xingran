/**
 * 极简现代风主题 - 浅色模式
 * 特点：干净线条、大量留白、高对比度、细边框
 */

import type { ThemeConfig } from "@/types/theme";
import { spacing } from "../../tokens/spacing";
import { fontFamily } from "../../tokens/typography";

export const minimalLight: ThemeConfig = {
  id: "minimal",
  name: "极简现代",
  description: "干净的线条、大量留白、高对比度，强调内容本身",

  colors: {
    // 主色：纯黑系列
    primary: ["#000000", "#1a1a1a", "#333333", "#4d4d4d", "#666666"],
    secondary: ["#f5f5f5", "#e8e8e8", "#d9d9d9", "#bfbfbf", "#a6a6a6"],
    accent: ["#000000", "#1a1a1a", "#333333"],
    neutral: [
      "#fafafa",
      "#f5f5f5",
      "#e8e8e8",
      "#d9d9d9",
      "#bfbfbf",
      "#8c8c8c",
      "#595959",
      "#262626",
      "#000000",
    ],

    // 功能色（使用纯色）
    success: ["#52c41a", "#389e0d", "#237804"],
    warning: ["#faad14", "#d48806", "#ad6800"],
    error: ["#ff4d4f", "#cf1322", "#a8071a"],
    info: ["#000000", "#1a1a1a", "#333333"],
    processing: ["#52c41a", "#389e0d", "#237804"],

    // 背景：纯白系列
    background: {
      primary: "#ffffff",
      secondary: "#fafafa",
      tertiary: "#f5f5f5",
      surface: "#ffffff",
      elevated: "#ffffff",
    },

    // 文字：高对比度
    text: {
      primary: "#000000",
      secondary: "#595959",
      tertiary: "#8c8c8c",
      disabled: "#bfbfbf",
      inverse: "#ffffff",
    },

    // 边框：细线
    border: {
      primary: "#e8e8e8",
      secondary: "#f0f0f0",
      divider: "#f0f0f0",
    },
  },

  spacing: {
    xs: spacing.xs,
    sm: spacing.sm,
    md: spacing.lg, // 更大的间距
    lg: spacing.xl, // 更大的间距
    xl: spacing["2xl"], // 更大的间距
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
    sm: "0 1px 2px rgba(0, 0, 0, 0.04)",
    md: "0 2px 4px rgba(0, 0, 0, 0.06)",
    lg: "0 4px 8px rgba(0, 0, 0, 0.08)",
    xl: "0 8px 16px rgba(0, 0, 0, 0.1)",
    "2xl": "0 12px 24px rgba(0, 0, 0, 0.12)",
    inner: "inset 0 1px 2px rgba(0, 0, 0, 0.04)",
  },

  radius: {
    sm: "2px", // 更小的圆角
    md: "4px", // 更小的圆角
    lg: "6px", // 更小的圆角
    xl: "8px", // 更小的圆角
    "2xl": "12px",
    full: "9999px",
  },

  effects: {
    minimal: {
      borderWidth: "1px",
      borderColor: "#e8e8e8",
    },
    transition: {
      fast: "150ms cubic-bezier(0.4, 0, 0.2, 1)",
      base: "200ms cubic-bezier(0.4, 0, 0.2, 1)",
      slow: "300ms cubic-bezier(0.4, 0, 0.2, 1)",
    },
  },
};
