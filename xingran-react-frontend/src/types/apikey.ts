/**
 * API 密钥管理相关类型
 */

import type { Status, PageParams } from "./base";

// ==================== API 密钥相关 ====================

/**
 * API 密钥类型
 */
export interface APIKey {
  id: string;
  name: string;
  key: string; // 脱敏显示（前12位）
  scopes: string[]; // read, write, admin
  ipWhitelist: string[];
  inheritPerms: boolean;
  expiresAt?: string;
  lastUsedAt?: string;
  isActive: boolean;
  description?: string;
  createdAt: string;
  updatedAt: string;
}

/**
 * 创建 API 密钥请求
 */
export interface CreateAPIKeyRequest {
  name: string;
  description?: string;
  scopes: string[];
  inheritPerms: boolean;
  ipWhitelist?: string[];
  expiresAt?: string;
}

/**
 * 更新 API 密钥请求
 */
export interface UpdateAPIKeyRequest {
  name?: string;
  description?: string;
  scopes?: string[];
  inheritPerms?: boolean;
  ipWhitelist?: string[];
  isActive?: boolean;
}

/**
 * API 密钥列表查询参数
 */
export interface APIKeyListParams extends PageParams {
  keyword?: string;
  status?: boolean;
  scope?: string;
  /** 服务端排序字段（对应后端 apiKeyAllowedSortFields 白名单 key） */
  orderByColumn?: string;
  /** 是否升序 */
  isAsc?: boolean;
}

// ==================== API 密钥使用日志相关 ====================

/**
 * API 密钥使用日志
 */
export interface APIKeyUsageLog {
  id: string;
  api_key_id: string;
  user_id: string;
  method: string;
  path: string;
  status_code: number;
  client_ip: string;
  ip_address?: string; // 别名，兼容性字段
  user_agent?: string;
  duration: number;
  success: boolean;
  created_at: string;
}

/**
 * 使用统计汇总
 */
export interface UsageSummary {
  total_requests: number;
  success_rate: number;
  avg_duration: number;
  requests_by_method: Record<string, number>;
  requests_by_path: Record<string, number>;
  errors_by_status: Record<number, number>;
}

// ==================== 类型别名 ====================

/**
 * 分页数据类型（从 base.ts 导入）
 */
export type PageData<T> = import("./base").PageResponse<T>;
