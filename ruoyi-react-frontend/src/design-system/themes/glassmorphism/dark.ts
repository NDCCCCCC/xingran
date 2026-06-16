/**
 * 玻璃拟态主题 - 深色模式
 * 特点：深色半透明背景、背景模糊、柔和光效、现代感
 * WCAG AA 对比度：文字 #E0E7FF 在 #0A0A14 上 ~12:1 ✓
 */

import type { ThemeConfig } from "@/types/theme";
import { spacing } from "../../tokens/spacing";
import { fontFamily } from "../../tokens/typography";
import { radius } from "../../tokens/shadows";

export const glassmorphismDark: ThemeConfig = {
	id: "glassmorphism",
	name: "玻璃拟态（深色）",
	description: "深色半透明背景、背景模糊效果，营造轻盈现代感",

	colors: {
		primary: ["#c7d2fe", "#a5b4fc", "#818cf8", "#6366f1", "#4f46e5"],
		secondary: ["#e9d5ff", "#d8b4fe", "#c084fc", "#a855f7", "#9333ea"],
		accent: ["#d8b4fe", "#c084fc", "#a855f7"],
		neutral: ["#1a1a2e", "#16162a", "#121226", "#0e0e22", "#0a0a1e", "#060616", "#020210", "#000008", "#000000"],

		success: ["#4ade80", "#22c55e", "#16a34a"],
		warning: ["#fbbf24", "#f59e0b", "#d97706"],
		error: ["#f87171", "#ef4444", "#dc2626"],
		info: ["#60a5fa", "#3b82f6", "#2563eb"],
		processing: ["#34d399", "#10b981", "#059669"],

		background: {
			primary: "rgba(10, 10, 20, 0.85)",
			secondary: "rgba(10, 10, 20, 0.65)",
			tertiary: "rgba(10, 10, 20, 0.45)",
			surface: "rgba(20, 20, 35, 0.75)",
			elevated: "rgba(30, 30, 50, 0.85)",
		},

		text: {
			primary: "#e0e7ff",
			secondary: "#c7d2fe",
			tertiary: "#a5b4fc",
			disabled: "#6366f1",
			inverse: "#0a0a14",
		},

		border: {
			primary: "rgba(255, 255, 255, 0.15)",
			secondary: "rgba(255, 255, 255, 0.08)",
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
		xs: "0 2px 8px rgba(0, 0, 0, 0.4)",
		sm: "0 4px 12px rgba(0, 0, 0, 0.5)",
		md: "0 8px 20px rgba(0, 0, 0, 0.6)",
		lg: "0 12px 32px rgba(0, 0, 0, 0.7)",
		xl: "0 20px 48px rgba(0, 0, 0, 0.75)",
		"2xl": "0 32px 64px rgba(0, 0, 0, 0.8)",
		inner: "inset 0 2px 8px rgba(0, 0, 0, 0.3)",
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
			opacity: "0.85",
			border: "1px solid rgba(255, 255, 255, 0.15)",
			saturation: "140%",
		},
		transition: {
			fast: "200ms cubic-bezier(0.4, 0, 0.2, 1)",
			base: "300ms cubic-bezier(0.4, 0, 0.2, 1)",
			slow: "500ms cubic-bezier(0.4, 0, 0.2, 1)",
		},
	},
};
