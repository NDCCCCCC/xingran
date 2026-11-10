/**
 * AD 同步漂移告警 API（Phase 144 ALRT-03）
 *
 * 全部走 `@/lib/api` 的 wrapped `post`（`Promise<BaseResponse<T>>`），
 * 直接返回 `result.data` —— 禁止裸 axios、禁止手动判 `code===0`（拦截器已处理）。
 *
 * 端点契约（PLAN-01-backend 实现）：
 *   POST /system/ad-drift-alert/{status,logs,config,config/update,trigger}
 *   - status : 状态卡数据（含累计告警次数 / 各通道开关 / 基线 / 上次告警）
 *   - logs   : 触发记录分页（list/total/current/pageSize，current 从 1）
 *   - config / config/update : 9 个 sys_config 旋钮读写
 *   - trigger: 立即检测（写操作，权限码 system:adDriftAlert:trigger）
 *
 * 字段名用 camelCase（与后端 `GetStatus` 响应及 `jsonResult` 口径一致）；
 * 括号内注释标注对应的 `sys_config` key，不把 key 当字段名。
 */
import { post } from "@/lib/api";
import type { PageResponse } from "@/types/base";

const BASE = "/system/ad-drift-alert";

/** 状态卡数据（status 端点响应 data） */
export interface ADDriftAlertStatus {
  /** 是否启用告警（sys_config: ad_sync.drift_alert.enabled） */
  enabled: boolean;
  /** 容差（任一计数变化超过该值才告警） */
  tolerance: number;
  /** 节流窗口（分钟） */
  throttleMinutes: number;
  /** 站内通知（sys_notice / 铃铛）开关 */
  noticeEnabled: boolean;
  /** WS 实时广播开关 */
  wsEnabled: boolean;
  /** webhook 推送开关 */
  webhookEnabled: boolean;
  /** 当前基线（member_count,distinct_groups 编码或数值，展示用） */
  baseline: number | string | null;
  /** 上次告警时间（ISO 字符串，无则 null） */
  lastAlertAt: string | null;
  /** 上次告警是否检测到漂移 */
  lastAlertDrifted: boolean;
  /** 累计告警次数 */
  alertCount: number;
}

/**
 * 触发记录（logs 端点 list 元素，来自 sys_oper_log 投影）。
 * B-1 / v1.56：与后端 DriftAlertTriggerLog 逐字段对齐（此前前端按虚构业务字段
 * 渲染，全部列落空 fallback）。noticeId / isRead 为后端派生字段（gorm:"-"），
 * 来自 JsonResult.noticeId + sys_notice_read 按用户已读状态。
 */
export interface ADDriftTriggerLog {
  id: string;
  /** 日志标题（恒为 "AD 同步漂移告警"） */
  title: string;
  /** 方法名（恒为 "ADSyncDriftAlertService.CheckAndAlert"） */
  method: string;
  /** 触发时间（字符串，dayjs 容错格式化） */
  operTime: string;
  /** 原始诊断 JSON（memberCount/distinctGroups/baseline/memberDelta/groupDelta/noticeId） */
  jsonResult: string;
  /** operlog 状态：0=成功 1=失败（非「漂移/节流」业务状态） */
  status: number;
  /** 关联的 sys_notice ID（无关联时为空串，不能标记已读） */
  noticeId: string;
  /** 当前用户是否已读该通知 */
  isRead: boolean;
}

/** 触发记录查询参数（current 从 1）。
 * B-1 / v1.56：后端 ListLogs 只接 current/pageSize，时间/状态筛选不存在，
 * 前端不再发送假参数（不伪造后端能力）。 */
export interface ADDriftTriggerLogParams {
  current: number;
  pageSize: number;
}

/** 9 个配置旋钮（config / config/update 的 data 形状） */
export interface ADDriftAlertConfig {
  /** 1. 启用告警（ad_sync.drift_alert.enabled） */
  enabled: boolean;
  /** 2. 容差（ad_sync.drift_alert.tolerance） */
  tolerance: number;
  /** 3. 节流分钟（ad_sync.drift_alert.throttle_minutes） */
  throttleMinutes: number;
  /** 4. 站内通知（ad_sync.drift_alert.notice_enabled） */
  noticeEnabled: boolean;
  /** 5. WS 实时广播（ad_sync.drift_alert.ws_enabled） */
  wsEnabled: boolean;
  /** 6. webhook 推送（ad_sync.drift_alert.webhook_enabled） */
  webhookEnabled: boolean;
  /** 7. webhook 配置 ID（空 = 用默认配置；ad_sync.drift_alert.webhook_config_id） */
  webhookConfigId: string;
  /** 8. webhook 模板（ad_sync.drift_alert.webhook_template） */
  webhookTemplate: string;
  /** 9. 基线展示（只读回显，ad_sync.drift_alert.baseline） */
  baseline: string;
}

/** 立即检测结果（trigger 端点实际返回 {message}，B-1 / v1.56 对齐） */
export interface ADDriftTriggerResult {
  message?: string;
}

/** 读取状态卡数据 */
export async function getStatus(): Promise<ADDriftAlertStatus> {
  const result = await post<ADDriftAlertStatus>(`${BASE}/status`);
  return result.data as ADDriftAlertStatus;
}

/** 读取触发记录（分页，current 从 1） */
export async function listTriggerLogs(
  params: ADDriftTriggerLogParams
): Promise<PageResponse<ADDriftTriggerLog>> {
  const result = await post<PageResponse<ADDriftTriggerLog>>(`${BASE}/logs`, params);
  return result.data as PageResponse<ADDriftTriggerLog>;
}

/** 读取 9 个配置旋钮 */
export async function getConfig(): Promise<ADDriftAlertConfig> {
  const result = await post<ADDriftAlertConfig>(`${BASE}/config`);
  return result.data as ADDriftAlertConfig;
}

/** 写 9 个配置旋钮（权限码 system:adDriftAlert:edit 门控） */
export async function updateConfig(payload: ADDriftAlertConfig): Promise<ADDriftAlertConfig> {
  const result = await post<ADDriftAlertConfig>(`${BASE}/config/update`, payload);
  return result.data as ADDriftAlertConfig;
}

/** 立即检测（写操作，权限码 system:adDriftAlert:trigger 门控） */
export async function triggerOnce(): Promise<ADDriftTriggerResult> {
  const result = await post<ADDriftTriggerResult>(`${BASE}/trigger`);
  return result.data as ADDriftTriggerResult;
}

/** 把某条触发记录对应的站内通知标记为当前用户已读（B-1 / v1.56）。
 * 后端 POST /logs/read 按 {logId} 反查 JsonResult.noticeId 后复用
 * NoticeService.MarkNoticeRead；noticeId 为空 / 记录不存在时后端返回 400，
 * 故调用方须对 isRead 或无 noticeId 的行禁用按钮。 */
export async function markTriggerLogRead(logId: string): Promise<{ message?: string }> {
  const result = await post<{ message?: string }>(`${BASE}/logs/read`, { logId });
  return result.data as { message?: string };
}
