/**
 * 静谧奢华主题 - 深色模式
 * 特点：深邃炭黑背景 + 铜金色点缀 + 优雅动效 + 奢华质感
 * WCAG AA 对比度：文字 #F0F2F5 在 #0A0E12 上 ~17:1 ✓
 */

import type { ThemeConfig } from "@/types/theme";
import { spacing } from "../../tokens/spacing";
import { fontFamily } from "../../tokens/typography";
import { radius } from "../../tokens/shadows";

export const luxuryQuietDark: ThemeConfig = {
	id: "luxury-quiet",
	name: "静谧奢华 (Dark)",
	description: "深邃炭黑背景配铜金色点缀，优雅奢华的视觉体验",

	colors: {
		primary: ["#fef3c7", "#fde68a", "#d4af37", "#b8960f", "#9a7b0a"],
		secondary: ["#f8f9fa", "#e9ecef", "#dee2e6", "#adb5bd", "#6c757d"],
		accent: ["#d4af37", "#f4d03f", "#b8960f"],
		neutral: ["#f8f9fa", "#e9ecef", "#dee2e6", "#ced4da", "#adb5bd", "#6c757d", "#495057", "#343a40", "#212529"],

		success: ["#73d13d", "#52c41a", "#389e0d"],
		warning: ["#ffd666", "#faad14", "#d48806"],
		error: ["#ff7875", "#ff4d4f", "#cf1322"],
		info: ["#69c0ff", "#1890ff", "#096dd9"],
		processing: ["#52c41a", "#389e0d", "#237804"],

		background: {
			primary: "#0a0e12",
			secondary: "#11151a",
			tertiary: "#1a1f26",
			surface: "#1a1f26",
			elevated: "#222830",
		},

		text: {
			primary: "#f0f2f5",
			secondary: "#b8c5d6",
			tertiary: "#7d8ca1",
			disabled: "#4a5568",
			inverse: "#0a0e12",
		},

		border: {
			primary: "rgba(212, 175, 55, 0.3)",
			secondary: "rgba(184, 149, 15, 0.2)",
			divider: "rgba(255, 255, 255, 0.06)",
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
