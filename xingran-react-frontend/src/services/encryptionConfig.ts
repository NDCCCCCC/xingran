/**
 * 加密配置服务
 * 提供加密配置的获取、缓存和刷新功能
 */

import { get } from "@/lib/api";

export interface EncryptionConfig {
  enabled: boolean;
  key: string;
  source: string;
}

// 缓存加密配置（5分钟TTL）
let encryptionConfigCache: EncryptionConfig | null = null;
let cacheTimestamp = 0;
const CACHE_TTL = 5 * 60 * 1000; // 5分钟

/**
 * 获取加密配置（公共端点，无需认证）
 */
export async function getEncryptionConfig(): Promise<EncryptionConfig> {
  const response = await get<EncryptionConfig>("/system/auth/encryption-config");
  return response.data!;
}

/**
 * 获取缓存的加密配置（如果缓存未过期）
 */
export async function getCachedEncryptionConfig(): Promise<EncryptionConfig> {
  const now = Date.now();

  // 缓存有效，直接返回
  if (encryptionConfigCache && (now - cacheTimestamp) < CACHE_TTL) {
    return encryptionConfigCache;
  }

  // 缓存过期或未加载，重新获取
  const config = await getEncryptionConfig();
  encryptionConfigCache = config;
  cacheTimestamp = now;

  return config;
}

/**
 * 清除加密配置缓存（配置更新后调用）
 */
export function clearEncryptionConfigCache(): void {
  encryptionConfigCache = null;
  cacheTimestamp = 0;
}

export default {
  getEncryptionConfig,
  getCachedEncryptionConfig,
  clearEncryptionConfigCache,
};
