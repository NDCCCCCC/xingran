/**
 * 扁平化2.0主题导出
 */

import type { ThemeType, ThemeConfig } from "@/types/theme";
import { flat2Light } from "./light";
import { flat2Dark } from "./dark";

export { flat2Light } from "./light";
export { flat2Dark } from "./dark";

/**
 * 主题变体
 */
export const flat2ThemeVariants = {
	light: flat2Light,
	dark: flat2Dark,
};

/**
 * 主题元数据
 */
export const flat2Metadata = {
	id: "flat2.0" as ThemeType,
	name: "扁平化2.0",
	nameEn: "Flat Design 2.0",
	description: "扁平设计配微妙渐变和阴影，鲜艳配色充满活力",
	tags: ["flat", "vibrant", "modern", "gradient", "energetic"],
	preview: {
		light: "linear-gradient(135deg, #667eea 0%, #764ba2 100%)",
		dark: "linear-gradient(135deg, #4c51bf 0%, #553c9a 100%)",
	},
	supportsModes: ["light", "dark"] as const,
};

/**
 * 获取指定模式的主题配置
 */
export function getFlat2Theme(mode: "light" | "dark" = "light"): ThemeConfig {
	return mode === "dark" ? flat2Dark : flat2Light;
}
