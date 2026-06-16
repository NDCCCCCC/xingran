/**
 * 静谧奢华主题导出
 */

import type { ThemeType, ThemeConfig } from "@/types/theme";
import { luxuryQuietLight } from "./light";
import { luxuryQuietDark } from "./dark";

/**
 * 主题变体
 */
export const luxuryQuietVariants = {
	light: luxuryQuietLight,
	dark: luxuryQuietDark,
};

/**
 * 主题元数据
 */
export const luxuryQuietMetadata = {
	id: "luxury-quiet" as ThemeType,
	name: "静谧奢华",
	nameEn: "Luxury Quiet",
	description: "深邃色调 + 金属点缀 + 优雅动效 + 奢华质感",
	tags: ["luxury", "elegant", "premium", "quiet"],
	preview: {
		light: "linear-gradient(135deg, #fafafa 0%, #f5f5f5 100%)",
		dark: "linear-gradient(135deg, #0a0e12 0%, #11151a 100%)",
	},
	supportsModes: ["light", "dark"] as const,
};

/**
 * 获取指定模式的主题配置
 */
export function getLuxuryQuietTheme(mode: "light" | "dark" = "light"): ThemeConfig {
	return mode === "dark" ? luxuryQuietDark : luxuryQuietLight;
}
