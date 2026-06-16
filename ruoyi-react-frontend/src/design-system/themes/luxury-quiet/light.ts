/**
 * 静谧奢华主题 - 浅色模式
 * 特点：柔和背景 + 铜金色点缀 + 奢华质感 + 优雅动效
 */

import type { ThemeConfig } from "@/types/theme";
import { spacing } from "../../tokens/spacing";
import { fontFamily } from "../../tokens/typography";
import { radius } from "../../tokens/shadows";

export const luxuryQuietLight: ThemeConfig = {
	id: "luxury-quiet",
	name: "静谧奢华",
	description: "柔和背景配铜金色点缀，优雅奢华的视觉体验",

	colors: {
		// 主色：铜金色系列
		primary: [
			"#fef3c7", // 50 - 浅金
			"#fde68a", // 100 - 浅金黄
			"#d4af37", // 200 - 铜金色（主题色）
			"#b8960f", // 300 - 深金
			"#9a7b0a", // 400 - 古铜
		],
		secondary: ["#f8f9fa", "#e9ecef", "#dee2e6", "#adb5bd", "#6c757d"],
		accent: ["#d4af37", "#f4d03f", "#b8960f"],
		neutral: [
			"#fafafa", "#f5f5f5", "#e8e8e8", "#d9d9d9", "#bfbfbf",
			"#8c8c8c", "#595959", "#374151", "#1a1f2e"
		],

		// 功能色 - 柔和版本
		success: ["#73d13d", "#52c41a", "#389e0d"],
		warning: ["#ffd666", "#faad14", "#d48806"],
		error: ["#ff7875", "#ff4d4f", "#cf1322"],
		info: ["#69c0ff", "#1890ff", "#096dd9"],
		processing: ["#52c41a", "#389e0d", "#237804"],

		// 背景：柔和浅色系列
		background: {
			primary: "#fafafa",    // 柔和浅灰白
			secondary: "#f5f5f5",  // 次级背景
			tertiary: "#ebebeb",   // 三级背景
			surface: "#ffffff",    // 表面背景
			elevated: "#ffffff",   // 悬浮背景
		},

		// 文字：深色系列
		text: {
			primary: "#1a1f2e",    // 主要文字 - 深炭色
			secondary: "#374151",  // 次要文字 - 深灰
			tertiary: "#6b7280",   // 三级文字 - 中灰
			disabled: "#9ca3af",   // 禁用文字
			inverse: "#ffffff",    // 反色文字
		},

		// 边框：金色半透明
		border: {
			primary: "rgba(212, 175, 55, 0.3)",     // 金色半透明
			secondary: "rgba(212, 175, 55, 0.15)",  // 浅金色
			divider: "rgba(0, 0, 0, 0.06)",         // 分割线
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
