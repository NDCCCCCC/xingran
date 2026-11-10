/**
 * AD 同步漂移告警 API（Phase 144 ALRT-03 / quick 261110-jci 契约对齐）
 *
 * 全部走 `@/lib/api` 的 wrapped `post`（`Promise<BaseResponse<T>>`），
 * 直接返回 `result.data` —— 禁止裸 axios、禁止手动判 `code===0`（拦截器已处理）。
 *
 * 端点契约（后端实现）：
 *   POST /system/ad-drift-alert/{status,logs,config,config/update,trigger,logs/read}
 *   - status : 状态卡数据（含累计告警次数 / 各通道开关 / 基线 / 上次告警）
 *   - logs   : 触发记录分页（list/total/current/pageSize，current 从 1）
 *   - config / config/update : 9 个 sys_config 旋钮读写，**dotted-key + 全 string**
 *   - trigger: 立即检测（写操作，权限码 system:adDriftAlert:trigger）
 *   - logs/read : 标记某条触发记录对应站内通知已读
 *
 * config 契约（B-1 / quick 261110-jci）：后端 `GetConfig` 返回
 * `DriftAlertConfigView = map[string]string`（key 为 `ad_sync.drift_alert.*`，
 * 值全为字符串），`UpdateConfig` 用 `ShouldBindJSON(&map[string]string)` +
 * 白名单消费同一形状。前端页面用 camelCase 类型化对象，故在**本文件内**做
 * 双向适配（不污染页面、不改后端）：`parseConfigView` 读、`toConfigPayload` 写。
 * 后端不存在 `webhook_template` / `webhookTemplate` 旋钮（死控件已删）。
 */
import { post } from "@/lib/api";
import type { PageResponse } from "@/types/base";

const BASE = "/system/ad-drift-alert";

/**
 * 9 个 sys_config 旋钮的 dotted key（与后端
 * `internal/services/addomain/ad_sync_drift_alert_query.go` 的
 * `adDriftConfigKeys` 逐字一致；本文件是前端唯一映射入口）。
 */
export const AD_DRIFT_CONFIG_KEYS = {
  enabled: "ad_sync.drift_alert.enabled",
  tolerance: "ad_sync.drift_alert.tolerance",
  throttleMinutes: "ad_sync.drift_alert.throttle_minutes",
  noticeEnabled: "ad_sync.drift_alert.notice_enabled",
  wsEnabled: "ad_sync.drift_alert.ws_enabled",
  webhookEnabled: "ad_sync.drift_alert.webhook_enabled",
  webhookConfigId: "ad_sync.drift_alert.webhook_config_id",
  baseline: "ad_sync.drift_alert.baseline",
  baselineKey: "ad_sync.drift_alert.baseline_key",
} as const;

/** 后端 `GET /config` 原始响应形状：9 个 dotted key，值全为 string。 */
export type ADDriftAlertConfigView = Record<string, string>;

/** 字符串转 boolean（仅 'true' 为真，其余含空串/非法值均为 false）。 */
function toBool(v: string | undefined): boolean {
  return v === "true";
}

/** 字符串转非负整数（空串 / 非法 / 负数回落 0，与后端 loadConfig 兜底同口径）。 */
function toNonNegInt(v: string | undefined): number {
  const n = Number(v);
  return Number.isFinite(n) && n >= 0 ? Math.trunc(n) : 0;
}

/**
 * 把后端 dotted-key map 还原成页面用的 camelCase 类型化对象。
 * `baseline` 是运行时基线（只读回显），`baselineKey` 仅 seed 占位不进表单。
 */
export function parseConfigView(
  view: ADDriftAlertConfigView | null | undefined
): ADDriftAlertConfig {
  const v = view ?? {};
  return {
    enabled: toBool(v[AD_DRIFT_CONFIG_KEYS.enabled]),
    tolerance: toNonNegInt(v[AD_DRIFT_CONFIG_KEYS.tolerance]),
    throttleMinutes: toNonNegInt(v[AD_DRIFT_CONFIG_KEYS.throttleMinutes]),
    noticeEnabled: toBool(v[AD_DRIFT_CONFIG_KEYS.noticeEnabled]),
    wsEnabled: toBool(v[AD_DRIFT_CONFIG_KEYS.wsEnabled]),
    webhookEnabled: toBool(v[AD_DRIFT_CONFIG_KEYS.webhookEnabled]),
    webhookConfigId: v[AD_DRIFT_CONFIG_KEYS.webhookConfigId] ?? "",
    baseline: v[AD_DRIFT_CONFIG_KEYS.baseline] ?? "",
  };
}

/**
 * 把 camelCase 表单对象转成后端 `UpdateConfig` 期望的 dotted-key 全 string map。
 * 只发后端白名单内的 9 个 key；`baseline` 只读回显也带值（白名单含该 key），
 * 不再发后端不存在的 `webhookTemplate` 或任何 camelCase 字段。
 */
export function toConfigPayload(config: ADDriftAlertConfig): ADDriftAlertConfigView {
  return {
    [AD_DRIFT_CONFIG_KEYS.enabled]: String(config.enabled),
    [AD_DRIFT_CONFIG_KEYS.tolerance]: String(config.tolerance),
    [AD_DRIFT_CONFIG_KEYS.throttleMinutes]: String(config.throttleMinutes),
    [AD_DRIFT_CONFIG_KEYS.noticeEnabled]: String(config.noticeEnabled),
    [AD_DRIFT_CONFIG_KEYS.wsEnabled]: String(config.wsEnabled),
    [AD_DRIFT_CONFIG_KEYS.webhookEnabled]: String(config.webhookEnabled),
    [AD_DRIFT_CONFIG_KEYS.webhookConfigId]: config.webhookConfigId ?? "",
    [AD_DRIFT_CONFIG_KEYS.baseline]: config.baseline ?? "",
    [AD_DRIFT_CONFIG_KEYS.baselineKey]: AD_DRIFT_CONFIG_KEYS.baseline,
  };
}

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

/** 9 个配置旋钮对应的表单/页面 camelCase 形状（由 parseConfigView 产出）。
 * 注意：后端没有 webhookTemplate 旋钮，该字段已删（quick 261110-jci）。 */
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
  /** 8. 基线展示（只读回显，ad_sync.drift_alert.baseline；随提交原样带回） */
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

/** 读取 9 个配置旋钮：后端返回 dotted-key map，转成 camelCase 类型化对象。 */
export async function getConfig(): Promise<ADDriftAlertConfig> {
  const result = await post<ADDriftAlertConfigView>(`${BASE}/config`);
  return parseConfigView(result.data as ADDriftAlertConfigView);
}

/** 写 9 个配置旋钮（权限码 system:adDriftAlert:edit 门控）。
 * 发 dotted-key 全 string map，与后端 ShouldBindJSON(&map[string]string)+白名单对齐；
 * 后端返回 {message}，据此回传合并后的表单对象供页面刷新。 */
export async function updateConfig(config: ADDriftAlertConfig): Promise<ADDriftAlertConfig> {
  const payload = toConfigPayload(config);
  await post<{ message?: string }>(`${BASE}/config/update`, payload);
  return config;
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
