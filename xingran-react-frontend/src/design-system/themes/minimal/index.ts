/**
 * 极简现代主题导出
 */

import type { ThemeType, ThemeConfig } from "@/types/theme";
import { minimalLight } from "./light";
import { minimalDark } from "./dark";

/**
 * 主题变体
 */
export const minimalThemeVariants = {
  light: minimalLight,
  dark: minimalDark,
};

/**
 * 主题元数据
 */
export const minimalMetadata = {
  id: "minimal" as ThemeType,
  name: "极简现代",
  nameEn: "Minimal Modern",
  description: "干净的线条、大量留白、高对比度",
  tags: ["minimal", "clean", "modern", "professional"],
  preview: {
    light: "linear-gradient(135deg, #ffffff 0%, #f5f5f5 100%)",
    dark: "linear-gradient(135deg, #0a0a0a 0%, #171717 100%)",
  },
  supportsModes: ["light", "dark"] as const,
};

/**
 * 获取指定模式的主题配置
 */
export function getMinimalTheme(mode: "light" | "dark" = "light"): ThemeConfig {
  return mode === "dark" ? minimalDark : minimalLight;
}
