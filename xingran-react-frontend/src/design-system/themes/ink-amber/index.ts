/**
 * 墨绿琥珀主题导出
 */

import type { ThemeType, ThemeConfig } from "@/types/theme";
import { inkAmberLight } from "./light";
import { inkAmberDark } from "./dark";

/**
 * 主题变体
 */
export const inkAmberVariants = {
  light: inkAmberLight,
  dark: inkAmberDark,
};

/**
 * 主题元数据
 */
export const inkAmberMetadata = {
  id: "ink-amber" as ThemeType,
  name: "墨绿琥珀",
  nameEn: "Ink Amber",
  description: "墨绿基调配琥珀金点缀，源自登录页品牌配色",
  tags: ["ink-green", "amber", "brand", "elegant", "login"],
  preview: {
    light: "linear-gradient(135deg, #14532d 0%, #d4a574 100%)",
    dark: "linear-gradient(135deg, #0f1512 0%, #8a6534 100%)",
  },
  supportsModes: ["light", "dark"] as const,
};

/**
 * 获取指定模式的主题配置
 */
export function getInkAmberTheme(mode: "light" | "dark" = "light"): ThemeConfig {
  return mode === "dark" ? inkAmberDark : inkAmberLight;
}
