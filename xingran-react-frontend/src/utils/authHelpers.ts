/**
 * 认证相关辅助函数
 * 封装 token 获取和请求头构建逻辑
 */

import { getTokenManager } from "@/store/authStore";
import {
  getCachedEncryptionConfig,
  clearEncryptionConfigCache,
} from "@/services/encryptionConfig";
import type { EncryptionConfig } from "@/services/encryptionConfig";

/**
 * 获取包含认证信息的请求头
 * @returns 包含 Authorization 的请求头对象
 */
export async function getAuthHeaders(): Promise<Record<string, string>> {
  const tokenManager = getTokenManager();
  const token = await tokenManager.getAccessToken();
  return token ? { "Authorization": `Bearer ${token}` } : {};
}

/**
 * 获取当前访问令牌
 * @returns 访问令牌字符串，如果不存在则返回空字符串
 */
export async function getAccessToken(): Promise<string> {
  const tokenManager = getTokenManager();
  return await tokenManager.getAccessToken();
}

/**
 * 为 fetch 请求添加认证头
 * @param url - 请求 URL
 * @param options - fetch 选项（可选）
 * @returns 添加了认证头的 fetch 选项
 */
export async function withAuth(
  url: string,
  options?: RequestInit
): Promise<[string, RequestInit]> {
  const headers = await getAuthHeaders();
  const mergedOptions: RequestInit = {
    ...options,
    headers: {
      ...(options?.headers as Record<string, string>),
      ...headers,
    },
  };

  return [url, mergedOptions];
}

/**
 * 刷新加密配置
 *
 * 清除本地缓存并从服务器获取最新的加密配置。
 * 用于配置管理页面手动刷新配置，或在配置变更后立即同步。
 *
 * @returns 最新的加密配置对象
 * @throws 如果获取配置失败（网络错误、服务器错误等）
 *
 * @example
 * ```typescript
 * try {
 *   const config = await refreshEncryptionConfig();
 *   console.log('加密配置已更新:', config.enabled);
 * } catch (error) {
 *   console.error('刷新加密配置失败:', error);
 * }
 * ```
 */
export async function refreshEncryptionConfig(): Promise<EncryptionConfig> {
  // 清除缓存
  clearEncryptionConfigCache();

  // 重新获取配置（这次会绕过缓存，直接请求服务器）
  const config = await getCachedEncryptionConfig();

  return config;
}

/**
 * 获取加密配置状态
 *
 * 返回当前加密配置的启用状态。这是一个便捷函数，用于只需要知道
 * 加密是否启用的场景（如条件渲染、逻辑判断等）。
 *
 * @returns 加密启用状态（true = 启用，false = 禁用）
 *
 * @example
 * ```typescript
 * if (await getEncryptionConfigStatus()) {
 *   console.log('请求加密已启用');
 * } else {
 *   console.log('请求加密已禁用');
 * }
 * ```
 */
export async function getEncryptionConfigStatus(): Promise<boolean> {
  try {
    const config = await getCachedEncryptionConfig();
    return config.enabled;
  } catch (error) {
    // 获取配置失败时，默认返回 true（启用加密）
    // 这是 fail-safe 策略：宁可加密也不要暴露敏感数据
    console.warn("[authHelpers] 获取加密配置失败，默认启用加密:", error);
    return true;
  }
}
