/**
 * 玻璃拟态主题导出
 */

import type { ThemeType, ThemeConfig } from "@/types/theme";
import { glassmorphismLight } from "./light";
import { glassmorphismDark } from "./dark";

/**
 * 主题变体
 */
export const glassmorphismThemeVariants = {
  light: glassmorphismLight,
  dark: glassmorphismDark,
};

/**
 * 主题元数据
 */
export const glassmorphismMetadata = {
  id: "glassmorphism" as ThemeType,
  name: "玻璃拟态",
  nameEn: "Glassmorphism",
  description: "半透明背景、背景模糊效果，营造轻盈现代感",
  tags: ["glass", "modern", "transparent", "blur", "frosted"],
  preview: {
    light: "linear-gradient(135deg, rgba(255, 255, 255, 0.75) 0%, rgba(245, 240, 255, 0.75) 100%)",
    dark: "linear-gradient(135deg, rgba(10, 10, 20, 0.85) 0%, rgba(20, 20, 35, 0.75) 100%)",
  },
  supportsModes: ["light", "dark"] as const,
};

/**
 * 获取指定模式的主题配置
 */
export function getGlassmorphismTheme(mode: "light" | "dark" = "light"): ThemeConfig {
  return mode === "dark" ? glassmorphismDark : glassmorphismLight;
}
