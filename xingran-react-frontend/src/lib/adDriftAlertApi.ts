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

/** 触发记录（logs 端点 list 元素，来自 sys_oper_log） */
export interface ADDriftTriggerLog {
  id: string;
  /** 触发时间 */
  createdAt: string;
  /** 操作人 */
  operator?: string;
  /** 是否检测到漂移 */
  drifted: boolean;
  /** 成员数 */
  memberCount?: number;
  /** 去重组数 */
  distinctGroups?: number;
  /** 漂移量摘要 */
  delta?: string;
  /** 是否被节流 */
  throttled?: boolean;
  /** 结果 / 状态描述 */
  result?: string;
  /** 触发来源（cron / manual） */
  source?: string;
}

/** 触发记录查询参数（current 从 1） */
export interface ADDriftTriggerLogParams {
  current: number;
  pageSize: number;
  /** 起始时间（含） */
  startTime?: string;
  /** 结束时间（含） */
  endTime?: string;
  /** 状态筛选：all / drifted / normal / throttled */
  status?: "all" | "drifted" | "normal" | "throttled";
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

/** 立即检测结果（trigger 端点响应 data，形状由后端定义，宽松声明） */
export interface ADDriftTriggerResult {
  drifted?: boolean;
  alerted?: boolean;
  throttled?: boolean;
  memberCount?: number;
  distinctGroups?: number;
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
export async function updateConfig(
  payload: ADDriftAlertConfig
): Promise<ADDriftAlertConfig> {
  const result = await post<ADDriftAlertConfig>(`${BASE}/config/update`, payload);
  return result.data as ADDriftAlertConfig;
}

/** 立即检测（写操作，权限码 system:adDriftAlert:trigger 门控） */
export async function triggerOnce(): Promise<ADDriftTriggerResult> {
  const result = await post<ADDriftTriggerResult>(`${BASE}/trigger`);
  return result.data as ADDriftTriggerResult;
}
