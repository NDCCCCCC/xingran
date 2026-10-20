import { del, get, post } from "./api";
import { createResourceApi } from "./apiFactory";
import type { PageParams } from "@/types";
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
import { getTokenStorage } from "@/store/authStore";

// ==================== 管理端通知 API ====================

// Phase 94 D-05: 标准 CRUD 形状函数委托共享工厂（导出签名零变化；
// GET 动词端点 / batch-delete / publish / withdraw / 用户端 my-notices 全族保持原样）
const adminNoticeCrud = createResourceApi<Notice>({ basePath: "/system/notices" });

/**
 * 获取通知列表（管理端）
 */
export function getNoticeList(
  params: NoticeListParams
): Promise<BaseResponse<PageResponse<Notice>>> {
  return adminNoticeCrud.list(params as unknown as PageParams & Record<string, unknown>);
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
  return adminNoticeCrud.get(id);
}

/**
 * 创建通知
 */
export function createNotice(
  data: CreateNoticeRequest
): Promise<BaseResponse<{ id: string; message: string }>> {
  return adminNoticeCrud.create(data) as Promise<BaseResponse<{ id: string; message: string }>>;
}

/**
 * 更新通知
 */
export function updateNotice(
  id: string,
  data: UpdateNoticeRequest
): Promise<BaseResponse<{ message: string }>> {
  return adminNoticeCrud.update(id, data) as Promise<BaseResponse<{ message: string }>>;
}

/**
 * 删除通知
 */
export function deleteNotice(id: string): Promise<BaseResponse<{ message: string }>> {
  return adminNoticeCrud.delete(id) as Promise<BaseResponse<{ message: string }>>;
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
  // WS 端点与 REST 同挂 /api/v1 前缀下（后端: /api/v1/system/ws/notices，
  // router.go: system→authorized→ws→SetupNoticeWebSocketRouter 的 /notices）。
  // ⚠ 不得剥掉 /api/vN 后缀 —— 旧逻辑剥掉后拼出 /system/ws/notices，
  // 握手 404 → 无限重连（生产实测 2026-10-20）。
  const apiBaseURL = import.meta.env.VITE_API_BASE_URL || "/api/v1";
  let wsBaseURL = import.meta.env.VITE_WS_BASE_URL || apiBaseURL.replace(/^http/, "ws");
  // ⚠ 必须经 authStore 的全局单例读 token:SecureTokenStorageImpl 的
  // accessToken 只存实例内存字段,本模块旧代码自建实例 → 永远读不到
  // TokenManager 写入的值 → WS 握手 token 恒为空(登录态也一样)。
  const token = getTokenStorage().getAccessToken();

  // 未登录(无 token)时返回空串:后端对空 token 的 WS 握手直接 401,
  // 拿不到合法会话就不该发起连接 —— 配合 useWebSocket 的空 URL 守卫,
  // 避免未登录页面上铃铛无限重连风暴(生产实测 2026-10-20)。
  if (!token) {
    return "";
  }

  // WebSocket 构造器只接受绝对 URL（ws:// / wss://）；API base 是
  // 根相对路径（如 "/api/v1" → 推导出 "/xingran"，或整段剥空得 ""）时必须
  // 用当前页面的协议与主机补全，否则 new WebSocket(relative) 直接抛
  // DOMException（dev 下 VITE_API_BASE_URL 带 http:// 前缀所以从未暴露）。
  // ⚠ 空串也算相对："/api/v1".replace(...) 的结果是 ""，不是 "/"。
  if (wsBaseURL === "" || wsBaseURL.startsWith("/")) {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    wsBaseURL = `${protocol}//${window.location.host}${wsBaseURL}`;
  }

  // 通过 query 参数传递 token
  return `${wsBaseURL}/system/ws/notices?token=${token || ""}`;
}
