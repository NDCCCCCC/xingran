/**
 * 认证相关辅助函数
 * 封装 token 获取和请求头构建逻辑
 */

import { getTokenManager } from '@/store/authStore';

/**
 * 获取包含认证信息的请求头
 * @returns 包含 Authorization 的请求头对象
 */
export async function getAuthHeaders(): Promise<Record<string, string>> {
  const tokenManager = getTokenManager();
  const token = await tokenManager.getAccessToken();
  return token ? { 'Authorization': `Bearer ${token}` } : {};
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
export async function withAuth<T extends RequestInit = RequestInit>(
  url: string,
  options?: T
): Promise<[string, T]> {
  const headers = await getAuthHeaders();
  const mergedOptions: T = {
    ...options,
    headers: {
      ...(options?.headers as Record<string, string>),
      ...headers,
    } as T['headers'],
  };

  return [url, mergedOptions];
}
