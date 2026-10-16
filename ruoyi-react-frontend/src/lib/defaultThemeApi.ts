/**
 * 默认主题配置 API
 * Default Theme Configuration API
 *
 * 合并入仓后统一从 @/types/config 导入 ThemeConfiguration，
 * 避免 types/config.ts 与 lib/defaultThemeApi.ts 双重定义不一致。
 */

import { post, get } from "@/lib/api";
import type { ThemeConfiguration } from "@/types/config";

export type { ThemeConfiguration };

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
