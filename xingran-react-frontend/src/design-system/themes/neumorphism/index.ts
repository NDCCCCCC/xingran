/**
 * 新拟态主题导出
 */

import type { ThemeType, ThemeConfig } from "@/types/theme";
import { neumorphismLight } from "./light";
import { neumorphismDark } from "./dark";

/**
 * 主题变体
 */
export const neumorphismThemeVariants = {
  light: neumorphismLight,
  dark: neumorphismDark,
};

/**
 * 主题元数据
 */
export const neumorphismMetadata = {
  id: "neumorphism" as ThemeType,
  name: "新拟态",
  nameEn: "Neumorphism",
  description: "柔和的阴影浮雕效果，营造质感和深度",
  tags: ["neumorphism", "soft", "modern", "elegant", "3d", "rounded"],
  preview: {
    light: "linear-gradient(135deg, #e0e5ec 0%, #d1d9e6 100%)",
    dark: "linear-gradient(135deg, #1A1F2E 0%, #252B3D 100%)",
  },
  supportsModes: ["light", "dark"] as const,
};

/**
 * 获取指定模式的主题配置
 */
export function getNeumorphismTheme(mode: "light" | "dark" = "light"): ThemeConfig {
  return mode === "dark" ? neumorphismDark : neumorphismLight;
}
