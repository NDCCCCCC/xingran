/**
 * 默认主题配置 API
 * Default Theme Configuration API
 *
 * v1.22 Phase 65（D-01）：多主题移除后类型收窄为仅明暗模式；
 * 后端契约字段不动（旧 style/customColors 由后端继续忽略）。
 */

import { post, get } from "@/lib/api";

/**
 * 主题配置接口（Phase 65 收窄版：仅明暗模式）
 */
export interface ThemeConfiguration {
  mode: "light" | "dark";
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
