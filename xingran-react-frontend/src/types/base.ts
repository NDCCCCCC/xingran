/**
 * 基础类型定义
 */

/**
 * 通用API响应类型
 */
export interface BaseResponse<T = unknown> {
  code: number;
  message: string;
  data?: T;
  timestamp: number;
  request_id: string;
}

/**
 * 空响应类型
 */
export type EmptyResponse = BaseResponse<void>;

/**
 * 分页响应类型
 */
export type PaginatedResponse<T> = BaseResponse<PageResponse<T>>;

/**
 * 分页响应类型
 */
export interface PageResponse<T> {
  list: T[];
  total: number;
  current: number;
  pageSize: number;
}

/**
 * 分页查询参数
 */
export interface PageParams {
  current?: number;
  pageSize?: number;
}

/**
 * 状态类型：0=启用/正常，1=禁用/停用
 */
export type Status = 0 | 1;

/**
 * 性别：0=男，1=女，2=未知
 */
export type Gender = 0 | 1 | 2;
