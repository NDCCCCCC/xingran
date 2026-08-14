import { del, get, post } from "./api";
import type {
  Notice,
  NoticeListItem,
  NoticeStatistics,
  NoticeListParams,
  UserNoticeListParams,
  CreateNoticeRequest,
  UpdateNoticeRequest,
  PageResponse,
} from "@/types/notice";
import type { BaseResponse } from "@/types";
import { SecureTokenStorageImpl } from "@/utils/token/SecureTokenStorageImpl";

// 单例 token 存储，用于 WebSocket 连接同步获取 token
const tokenStorage = new SecureTokenStorageImpl();

// ==================== 管理端通知 API ====================

/**
 * 获取通知列表（管理端）
 */
export function getNoticeList(
  params: NoticeListParams
): Promise<BaseResponse<PageResponse<Notice>>> {
  return post("/system/notices/list", params);
}

/** 通知状态统计（按发布状态聚合，供统计卡片） */
export interface NoticeStatusStatistics {
  total: number;
  published: number;
  draft: number;
  scheduled: number;
}

/**
 * 获取通知状态统计（总数 / 已发布 / 草稿 / 定时发布）
 * 后端用 COUNT 聚合，不受 list 分页上限影响。
 */
export function getNoticeStatusStatistics(): Promise<BaseResponse<NoticeStatusStatistics>> {
  return post("/system/notices/statistics", {});
}

/**
 * 获取通知详情
 */
export function getNoticeDetail(id: string): Promise<BaseResponse<Notice>> {
  return post(`/system/notices/${id}`, {});
}

/**
 * 创建通知
 */
export function createNotice(
  data: CreateNoticeRequest
): Promise<BaseResponse<{ id: string; message: string }>> {
  return post("/system/notices", data);
}

/**
 * 更新通知
 */
export function updateNotice(
  id: string,
  data: UpdateNoticeRequest
): Promise<BaseResponse<{ message: string }>> {
  return post(`/system/notices/${id}/update`, data);
}

/**
 * 删除通知
 */
export function deleteNotice(id: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/system/notices/${id}/delete`, {});
}

/**
 * 批量删除通知
 */
export function batchDeleteNotices(ids: string[]): Promise<BaseResponse<{ message: string }>> {
  return post("/system/notices/batch-delete", { ids });
}

/**
 * 获取通知阅读统计
 */
export function getNoticeStatistics(id: string): Promise<BaseResponse<NoticeStatistics>> {
  return get(`/system/notices/${id}/statistics`);
}

/**
 * 发布通知
 */
export function publishNotice(id: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/system/notices/${id}/publish`, {});
}

/**
 * 撤回/取消发布通知
 */
export function withdrawNotice(id: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/system/notices/${id}/withdraw`, {});
}

// ==================== 用户端通知 API ====================

/**
 * 获取用户可见的通知列表
 */
export function getMyNotices(
  params: UserNoticeListParams = {}
): Promise<BaseResponse<PageResponse<Notice>>> {
  return get("/system/my-notices", params);
}

/**
 * 获取用户通知详情
 */
export function getMyNoticeDetail(id: string): Promise<BaseResponse<Notice>> {
  return get(`/system/my-notices/${id}`);
}

/**
 * 标记通知为已读
 */
export function markNoticeAsRead(id: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/system/my-notices/${id}/read`, {});
}

/**
 * 标记所有通知为已读
 */
export function markAllNoticesAsRead(): Promise<BaseResponse<{ message: string }>> {
  return post("/system/my-notices/read-all", {});
}

/**
 * 忽略通知（从列表中移除）
 */
export function ignoreNotice(id: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/system/my-notices/${id}/ignore`, {});
}

/**
 * 取消忽略通知（恢复显示）
 */
export function unignoreNotice(id: string): Promise<BaseResponse<{ message: string }>> {
  return del(`/system/my-notices/${id}/ignore`);
}

/**
 * 获取未读通知数量
 */
export function getUnreadCount(): Promise<BaseResponse<{ count: number }>> {
  return get("/system/my-notices/unread-count");
}

/**
 * 获取通知列表（用于铃铛下拉）
 */
export function getNotificationList(
  params: { current?: number; pageSize?: number } = {}
): Promise<BaseResponse<PageResponse<NoticeListItem>>> {
  return get("/system/my-notices", {
    current: params.current || 1,
    pageSize: params.pageSize || 10,
  });
}

// ==================== WebSocket ====================

/**
 * 构建 WebSocket 连接 URL
 */
export function buildWebSocketUrl(): string {
  // 与 VITE_API_BASE_URL 解耦：优先使用独立的 VITE_WS_BASE_URL，
  // 未设置时从 API base 推导（去掉 /api/vN 后缀并将 http(s) 转成 ws(s)），
  // 保持向后兼容，不再硬编码开发机 IP。
  const apiBaseURL = import.meta.env.VITE_API_BASE_URL || "/api/v1";
  const wsBaseURL =
    import.meta.env.VITE_WS_BASE_URL ||
    apiBaseURL.replace(/\/api\/v\d+$/, "").replace(/^http/, "ws");
  const token = tokenStorage.getAccessToken();

  // 通过 query 参数传递 token
  return `${wsBaseURL}/system/ws/notices?token=${token || ""}`;
}
