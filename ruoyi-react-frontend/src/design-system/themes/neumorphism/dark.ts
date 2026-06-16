/**
 * 新拟态主题 - 深色模式
 * 特点：深色柔和背景、微妙浮雕效果、圆润设计、同色调高低光
 * WCAG AA 对比度：文字 #E0E7FF 在 #1A1F2E 上 ~9:1 ✓
 */

import type { ThemeConfig } from "@/types/theme";
import { spacing } from "../../tokens/spacing";
import { fontFamily } from "../../tokens/typography";

const NEU_BG = "#1A1F2E";
const NEU_LIGHT = "rgba(255, 255, 255, 0.08)";
const NEU_DARK = "rgba(0, 0, 0, 0.4)";

export const neumorphismDark: ThemeConfig = {
	id: "neumorphism",
	name: "新拟态 (Dark)",
	description: "深色柔和背景、微妙阴影浮雕效果，营造夜间使用的质感与深度",

	colors: {
		primary: ["#e0e7ff", "#c7d2fe", "#a5b4fc", "#818cf8", "#6366f1"],
		secondary: ["#f5f3ff", "#ede9fe", "#ddd6fe", "#c4b5fd", "#a855f7"],
		accent: ["#818cf8", "#6366f1", "#4f46e5"],
		neutral: ["#1A1F2E", "#252B3D", "#303750", "#3D4663", "#4B5573", "#5A6588", "#6A7599", "#7A85AA", "#8A95BB"],

		success: ["#4ade80", "#22c55e", "#16a34a"],
		warning: ["#fbbf24", "#f59e0b", "#d97706"],
		error: ["#f87171", "#ef4444", "#dc2626"],
		info: ["#60a5fa", "#3b82f6", "#2563eb"],
		processing: ["#34d399", "#10b981", "#059669"],

		background: {
			primary: NEU_BG,
			secondary: "#252B3D",
			tertiary: "#303750",
			surface: NEU_BG,
			elevated: "#1F2636",
		},

		text: {
			primary: "#e0e7ff",
			secondary: "#c7d2fe",
			tertiary: "#a5b4fc",
			disabled: "#6B7280",
			inverse: "#1A1F2E",
		},

		border: {
			primary: NEU_BG,
			secondary: "#252B3D",
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
		sm: "12px",
		md: "16px",
		lg: "20px",
		xl: "24px",
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
