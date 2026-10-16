// ==================== 通知公告系统增强类型 ====================

/**
 * 通知类型
 * 1: 公告
 * 2: 警告
 */
export type NoticeType = "1" | "2";

/**
 * 通知优先级
 * 0: 普通
 * 1: 重要
 * 2: 紧急
 */
export type NoticePriority = 0 | 1 | 2;

/**
 * 发布状态
 * 0: 已发布
 * 1: 草稿
 * 2: 已撤回
 * 3: 定时发布
 */
export type PublishStatus = 0 | 1 | 2 | 3;

/**
 * 目标类型
 * 0: 全部用户
 * 1: 指定部门
 * 2: 指定角色
 * 3: 指定用户
 */
export type TargetType = 0 | 1 | 2 | 3;

/**
 * 执行类型
 */
export type ExecutionType = "once" | "recurring";

/**
 * 周期性配置
 */
export interface RecurrenceConfig {
  cronExpression: string; // Cron 表达式
  endDate?: string; // 结束日期（可选）
}

/**
 * 通知目标
 */
export interface NoticeTarget {
  id: string;
  noticeId: string;
  targetType: "dept" | "role" | "user";
  targetId: string;
  createdAt: string;
}

/**
 * 通知阅读记录
 */
export interface NoticeRead {
  id: string;
  noticeId: string;
  userId: string;
  readAt: string;
  readIp?: string;
}

/**
 * 通知附件
 */
export interface NoticeAttachment {
  id: string;
  noticeId: string;
  fileName: string;
  fileSize: number;
  fileType: string;
  filePath: string;
  uploadedBy: string;
  uploadedByName?: string;
  createdAt: string;
}

/**
 * 增强的通知公告
 */
export interface Notice {
  id: string;
  noticeTitle: string;
  noticeType: NoticeType;
  noticeContent: string;
  status: 0 | 1; // 0:正常 1:关闭
  priority: NoticePriority;
  publishTime?: string;
  publishStatus: PublishStatus;
  targetType: TargetType;
  createdByName?: string;
  isMarkdown: boolean;
  endDate?: string; // 周期性通知结束时间
  createdAt: string;
  updatedAt: string;
  // 关联数据
  targets?: NoticeTarget[];
  reads?: NoticeRead[];
  attachments?: NoticeAttachment[];
  channels?: NoticeChannelRequest[]; // 渠道配置
  // 用户端扩展字段
  isRead?: boolean;
  readAt?: string;
}

/**
 * 通知统计信息
 */
export interface NoticeStatistics {
  totalTargets: number;
  readCount: number;
  unreadCount: number;
  readRate: number;
}

/**
 * 通知渠道类型
 */
export type NotificationChannelType = "web" | "email" | "sms" | "api";

/**
 * 通知渠道配置请求
 */
export interface NoticeChannelRequest {
  channelType: NotificationChannelType;
  emailConfigId?: string;
  apiConfigId?: string;
  customRecipients?: string[]; // 自定义收件人列表（邮件地址或企微用户代码）
}

/**
 * 创建通知请求（管理端）
 */
export interface CreateNoticeRequest {
  noticeTitle: string;
  noticeType: NoticeType;
  noticeContent: string;
  priority?: NoticePriority;
  publishTime?: string;
  executionType?: ExecutionType;
  recurrenceConfig?: RecurrenceConfig;
  targetType: TargetType;
  targetDepts?: string[];
  targetRoles?: string[];
  targetUsers?: string[];
  channels?: NoticeChannelRequest[];
  isMarkdown?: boolean;
  status?: 0 | 1;
}

/**
 * 更新通知请求（管理端）
 */
export interface UpdateNoticeRequest {
  noticeTitle?: string;
  noticeType?: NoticeType;
  noticeContent?: string;
  priority?: NoticePriority;
  status?: 0 | 1;
  publishTime?: string;
  clearPublishTime?: boolean; // 是否清除定时发布时间
}

/**
 * 通知列表查询参数（管理端）
 */
export interface NoticeListParams {
  noticeTitle?: string;
  noticeType?: NoticeType;
  createTime?: string;
  current: number;
  pageSize: number;
  // 服务端排序参数（透传给后端 noticeAllowedSortFields 白名单）
  orderByColumn?: string;
  isAsc?: boolean;
}

/**
 * 用户通知列表查询参数
 */
export interface UserNoticeListParams {
  current?: number;
  pageSize?: number;
  status?: "read" | "unread" | "all";
}

/**
 * WebSocket 消息类型
 */
export type WebSocketMessageType =
  "new_notice" | "ping" | "pong" | "rpa_progress" | "rpa_completed" | "rpa_failed";

/**
 * WebSocket 通知消息
 */
export interface WebSocketNoticeMessage {
  type: WebSocketMessageType;
  data: {
    noticeId: string;
    noticeTitle: string;
    noticeType: NoticeType;
    priority: NoticePriority;
    isMarkdown: boolean;
    createdAt: string;
  } | null;
  timestamp: number;
  // RPA 进度消息支持
  content?: string; // JSON 字符串格式的 RPA 进度数据
}

/**
 * 通知列表项（用于铃铛弹窗显示）
 */
export interface NoticeListItem {
  id: string;
  noticeTitle: string;
  noticeType: NoticeType;
  priority: NoticePriority;
  isRead: boolean;
  createdAt: string;
  publishTime?: string;
}

/**
 * 通知中心状态
 */
export interface NoticeCenterState {
  // 未读数量
  unreadCount: number;
  // 通知列表（用于铃铛下拉）
  notifications: NoticeListItem[];
  // 加载状态
  loading: boolean;
  // WebSocket 连接状态
  wsConnected: boolean;
}

/**
 * 优先级对应的标签和颜色
 */
export const PRIORITY_LABELS: Record<NoticePriority, string> = {
  0: "普通",
  1: "重要",
  2: "紧急",
};

export const PRIORITY_COLORS: Record<NoticePriority, string> = {
  0: "default",
  1: "warning",
  2: "error",
};

/**
 * 通知类型对应的标签和颜色
 */
export const NOTICE_TYPE_LABELS: Record<NoticeType, string> = {
  "1": "公告",
  "2": "警告",
};

export const NOTICE_TYPE_COLORS: Record<NoticeType, string> = {
  "1": "blue",
  "2": "orange",
};

/**
 * 发布状态对应的标签和颜色
 */
export const PUBLISH_STATUS_LABELS: Record<PublishStatus, string> = {
  0: "草稿",
  1: "已发布",
  2: "定时发布中",
  3: "已撤回",
};

export const PUBLISH_STATUS_COLORS: Record<PublishStatus, string> = {
  0: "default",
  1: "success",
  2: "processing",
  3: "warning",
};

/**
 * 目标类型对应的标签
 */
export const TARGET_TYPE_LABELS: Record<TargetType, string> = {
  0: "全部用户",
  1: "指定部门",
  2: "指定角色",
  3: "指定用户",
};

/**
 * 角色选项（用于目标选择器）
 */
export interface RoleOption {
  id: string;
  roleName: string;
  roleKey: string;
}

/**
 * 用户选项（用于目标选择器）
 */
export interface UserOption {
  id: string;
  username: string;
  nickname?: string;
  deptName?: string;
}

// 重新导出 PageResponse
export type { PageResponse } from "./index";
