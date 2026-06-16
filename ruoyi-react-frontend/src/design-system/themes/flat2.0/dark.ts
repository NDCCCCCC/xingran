/**
 * 扁平化2.0主题 - 深色模式
 * 特点：深色背景、鲜艳配色、微妙渐变、清晰图标、充满活力
 * WCAG AA 对比度：文字 #E2E8F0 在 #0A0F1E 上 ~14:1 ✓
 */

import type { ThemeConfig } from "@/types/theme";
import { spacing } from "../../tokens/spacing";
import { fontFamily } from "../../tokens/typography";
import { radius } from "../../tokens/shadows";

export const flat2Dark: ThemeConfig = {
	id: "flat2.0",
	name: "扁平化2.0 (深色)",
	description: "深色背景配鲜艳渐变和阴影，活力十足的夜间模式",

	colors: {
		primary: ["#60a5fa", "#3b82f6", "#2563eb", "#1d4ed8", "#1e40af"],
		secondary: ["#a78bfa", "#8b5cf6", "#7c3aed", "#6d28d9", "#5b21b6"],
		accent: ["#f472b6", "#ec4899", "#db2777"],
		neutral: ["#f8fafc", "#f1f5f9", "#e2e8f0", "#cbd5e1", "#94a3b8", "#64748b", "#475569", "#334155", "#1e293b"],

		success: ["#4ade80", "#22c55e", "#16a34a"],
		warning: ["#fbbf24", "#f59e0b", "#d97706"],
		error: ["#f87171", "#ef4444", "#dc2626"],
		info: ["#60a5fa", "#3b82f6", "#2563eb"],
		processing: ["#34d399", "#10b981", "#059669"],

		background: {
			primary: "#0A0F1E",
			secondary: "#121827",
			tertiary: "#1A1F2E",
			surface: "#121827",
			elevated: "#1A1F2E",
		},

		text: {
			primary: "#e2e8f0",
			secondary: "#cbd5e1",
			tertiary: "#94a3b8",
			disabled: "#64748b",
			inverse: "#0a0f1e",
		},

		border: {
			primary: "#1e293b",
			secondary: "#334155",
			divider: "#1e293b",
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
		xs: "0 1px 2px rgba(0, 0, 0, 0.4)",
		sm: "0 2px 4px rgba(0, 0, 0, 0.5)",
		md: "0 4px 8px rgba(0, 0, 0, 0.6)",
		lg: "0 8px 16px rgba(0, 0, 0, 0.7)",
		xl: "0 12px 24px rgba(0, 0, 0, 0.75)",
		"2xl": "0 20px 40px rgba(0, 0, 0, 0.8)",
		inner: "inset 0 2px 4px rgba(0, 0, 0, 0.5)",
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
			gradient: "linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%)",
			hoverLift: "translateY(-2px)",
		},
		transition: {
			fast: "150ms cubic-bezier(0.4, 0, 0.2, 1)",
			base: "250ms cubic-bezier(0.4, 0, 0.2, 1)",
			slow: "400ms cubic-bezier(0.4, 0, 0.2, 1)",
		},
	},
};
