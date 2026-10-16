/**
 * API 密钥管理 API 客户端
 *
 * 该模块提供 API 密钥管理的所有前端 API 调用函数，包括：
 * - CRUD 操作：创建、查询、更新、删除 API 密钥
 * - 状态管理：启用/禁用 API 密钥
 * - 使用监控：查询使用日志和统计汇总
 *
 * 所有函数使用项目统一的 API 调用方式（@/lib/api.ts），
 * 自动处理认证、错误处理和响应格式化。
 *
 * @module api/apikey
 */

import { get, post } from "@/lib/api";
import type {
  APIKey,
  CreateAPIKeyRequest,
  UpdateAPIKeyRequest,
  APIKeyListParams,
  APIKeyUsageLog,
  UsageSummary,
  PageData,
} from "@/types/apikey";
import type { BaseResponse } from "@/types/base";

/**
 * 获取 API 密钥列表
 *
 * 支持分页查询、关键词搜索、状态筛选和作用域筛选。
 *
 * @param params - 查询参数（分页、关键词、状态、作用域）
 * @returns Promise 包含分页的 API 密钥列表
 *
 * @example
 * const result = await listAPIKeys({
 *   current: 1,
 *   pageSize: 10,
 *   keyword: 'production',
 *   status: true,
 *   scope: 'read'
 * });
 */
export function listAPIKeys(params?: APIKeyListParams): Promise<BaseResponse<PageData<APIKey>>> {
  return post("/system/apikeys/list", params);
}

/**
 * 创建 API 密钥
 *
 * 创建新的 API 密钥，自动生成密钥值。
 * 完整密钥值仅在创建时返回一次，请妥善保存。
 *
 * @param data - 创建请求参数（名称、作用域、IP 白名单等）
 * @returns Promise 包含完整密钥的响应（仅创建时返回一次）
 *
 * @example
 * const result = await createAPIKey({
 *   name: 'Production API Key',
 *   description: '用于生产环境的 API 访问',
 *   scopes: ['read', 'write'],
 *   inherit_perms: false,
 *   ip_whitelist: ['192.168.1.100', '10.0.0.0/24'],
 *   expires_at: '2025-12-31T23:59:59Z'
 * });
 * const fullKey = result.data.key; // 仅此一次返回完整密钥
 */
export function createAPIKey(data: CreateAPIKeyRequest): Promise<BaseResponse<{ key: string }>> {
  return post("/system/apikeys", data);
}

/**
 * 获取 API 密钥详情
 *
 * 根据密钥 ID 获取详细信息。
 * 注意：返回的密钥值为脱敏显示（仅前 12 位）。
 *
 * @param id - API 密钥 ID
 * @returns Promise 包含 API 密钥详情（脱敏）
 *
 * @example
 * const result = await getAPIKey('550e8400-e29b-41d4-a716-446655440000');
 * const apiKey = result.data;
 * console.log(apiKey.key); // 显示为 rec_1a2b3c4d5e6f...
 */
export function getAPIKey(id: string): Promise<BaseResponse<APIKey>> {
  return post(`/system/apikeys/${id}`);
}

/**
 * 更新 API 密钥
 *
 * 更新 API 密钥的配置信息（名称、描述、作用域、IP 白名单等）。
 * 不支持更新密钥值本身，如需更换密钥请删除后重新创建。
 *
 * @param id - API 密钥 ID
 * @param data - 更新请求参数（所有字段可选）
 * @returns Promise 空响应
 *
 * @example
 * await updateAPIKey('550e8400-e29b-41d4-a716-446655440000', {
 *   name: 'Updated API Key',
 *   scopes: ['read', 'write', 'admin'],
 *   ipWhitelist: ['10.0.0.0/8']
 * });
 */
export function updateAPIKey(id: string, data: UpdateAPIKeyRequest): Promise<BaseResponse<void>> {
  return post(`/system/apikeys/${id}/update`, data);
}

/**
 * 删除 API 密钥
 *
 * 软删除指定的 API 密钥，删除后无法恢复。
 * 建议在删除前先禁用密钥，确认无影响后再删除。
 *
 * @param id - API 密钥 ID
 * @returns Promise 空响应
 *
 * @example
 * await deleteAPIKey('550e8400-e29b-41d4-a716-446655440000');
 */
export function deleteAPIKey(id: string): Promise<BaseResponse<void>> {
  return post(`/system/apikeys/${id}/delete`);
}

/**
 * 切换 API 密钥状态（启用/禁用）
 *
 * 快速切换 API 密钥的启用状态，无需更新整个密钥配置。
 * 禁用的密钥将无法通过认证，但不会删除密钥本身。
 *
 * @param id - API 密钥 ID
 * @returns Promise 空响应
 *
 * @example
 * // 禁用密钥
 * await toggleAPIKeyStatus('550e8400-e29b-41d4-a716-446655440000');
 * // 再次调用可重新启用
 * await toggleAPIKeyStatus('550e8400-e29b-41d4-a716-446655440000');
 */
export function toggleAPIKeyStatus(id: string): Promise<BaseResponse<void>> {
  return post(`/system/apikeys/${id}/toggle`);
}

/**
 * 获取 API 密钥使用日志
 *
 * 查询指定 API 密钥的使用日志，记录每次 API 调用的详细信息。
 * 支持分页查询，按时间倒序排列。
 *
 * @param keyID - API 密钥 ID
 * @param params - 分页参数
 * @returns Promise 包含分页的使用日志列表
 *
 * @example
 * const result = await listUsageLogs('550e8400-e29b-41d4-a716-446655440000', {
 *   current: 1,
 *   pageSize: 20
 * });
 * const logs = result.data.list;
 * logs.forEach(log => {
 *   console.log(`${log.method} ${log.path} - ${log.status_code}`);
 * });
 */
export function listUsageLogs(
  keyID: string,
  params?: { current: number; pageSize: number }
): Promise<BaseResponse<PageData<APIKeyUsageLog>>> {
  return post(`/system/apikeys/${keyID}/logs`, params);
}

/**
 * 获取 API 密钥使用统计汇总
 *
 * 获取指定 API 密钥的使用统计汇总信息，包括：
 * - 总请求数
 * - 成功率
 * - 平均响应时间
 * - 按方法分组的请求数
 * - 按路径分组的请求数
 * - 按状态码分组的错误数
 *
 * @param keyID - API 密钥 ID
 * @returns Promise 包含统计汇总数据
 *
 * @example
 * const result = await getUsageSummary('550e8400-e29b-41d4-a716-446655440000');
 * const summary = result.data;
 * console.log(`总请求: ${summary.total_requests}`);
 * console.log(`成功率: ${(summary.success_rate * 100).toFixed(2)}%`);
 * console.log(`平均耗时: ${summary.avg_duration}ms`);
 */
export function getUsageSummary(keyID: string): Promise<BaseResponse<UsageSummary>> {
  return get(`/system/apikeys/${keyID}/summary`);
}
