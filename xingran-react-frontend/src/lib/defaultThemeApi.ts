/**
 * 默认主题配置 API
 * Default Theme Configuration API
 */

import { post, get } from "@/lib/api";

/**
 * 主题配置接口
 */
export interface ThemeConfiguration {
  mode: "light" | "dark" | "auto";
  style: "minimal" | "glassmorphism" | "neumorphism" | "flat2.0" | "luxury-quiet" | "ink-amber";
  customColors?: {
    primary?: string;
    sidebar?: string;
  };
}

/**
 * 获取默认主题配置
 */
export async function getDefaultThemeConfig(): Promise<ThemeConfiguration> {
  const result = await get<ThemeConfiguration>("/system/settings/config/theme/default");
  return result.data!;
}

/**
 * 设置默认主题配置
 */
export async function setDefaultThemeConfig(config: ThemeConfiguration): Promise<void> {
  await post("/system/settings/config/theme/default", config);
}

/**
 * 从用户配置同步到默认主题
 */
export async function syncUserThemeToDefault(userId: string): Promise<void> {
  await post("/system/settings/config/theme/sync", { user_id: userId });
}
