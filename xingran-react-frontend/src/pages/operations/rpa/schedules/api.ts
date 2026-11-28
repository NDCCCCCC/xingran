/**
 * 定时调度后端调用封装
 *
 * 约定（项目硬规则）：
 *   - 一律走 `@/lib/api` 的 post/get 包装函数，**禁裸 axios** —— 只有这样才会
 *     经过统一鉴权与 SM2/SM4 请求加密；
 *   - 后端路径与 Phase 158 已交付的 schedule_handler.go 严格对齐：
 *       POST /rpa/schedules/list        分页查询
 *       POST /rpa/schedules/:id          详情
 *       POST /rpa/schedules              创建
 *       POST /rpa/schedules/:id/update   更新
 *       POST /rpa/schedules/:id/delete   删除
 *       POST /rpa/schedules/:id/trigger  立即触发一次
 *       POST /rpa/schedules/:id/status   启停（0=启用 1=停用）
 */

import { post } from "@/lib/api";

/** 调度状态：0=启用 1=停用（与后端 rpamodels.Schedule.Status 同构） */
export const SCHEDULE_STATUS_ENABLED = 0;
export const SCHEDULE_STATUS_DISABLED = 1;

/** 调度记录（后端 sys_rpa_schedules 行的前端视图） */
export interface ScheduleRecord {
  id: string;
  /** 调度名称 */
  name: string;
  /** 关联的 RPA 任务标识（界面只展示任务名称，不展示该值） */
  taskId: string;
  /** 定时表达式（仅「高级设置」可见，界面主区域展示中文运行方式） */
  cronExpr: string;
  /** 时区 */
  timezone?: string;
  /** 生效开始时间 */
  startDate?: string | null;
  /** 生效结束时间 */
  endDate?: string | null;
  /** 状态：0=启用 1=停用 */
  status: number;
  /** 下次触发时间 */
  nextRunTime?: string | null;
  /** 最近触发时间 */
  lastRunTime?: string | null;
  /** 已触发次数 */
  runCount?: number;
  /** 备注说明 */
  description?: string;
  createdAt?: string;
  updatedAt?: string;
}

/** 列表查询参数（与后端 rpa.ScheduleListParams 同构） */
export interface ScheduleListParams {
  current: number;
  pageSize: number;
  /** 按名称模糊筛选 */
  name?: string;
  /** 按关联任务筛选 */
  taskId?: string;
  /** 按状态筛选：0=启用 1=停用 */
  status?: number;
}

/** 分页结果 */
export interface SchedulePage {
  list: ScheduleRecord[];
  total: number;
  current: number;
  pageSize: number;
}

/** 新建/编辑的提交载荷（name / taskId / cronExpr 必填） */
export interface SchedulePayload {
  name: string;
  taskId: string;
  cronExpr: string;
  timezone?: string;
  description?: string;
  status?: number;
}

const SCHEDULE_BASE = "/rpa/schedules";

/** 查询调度列表（分页） */
export async function listSchedules(params: ScheduleListParams): Promise<SchedulePage> {
  const result = await post<Partial<SchedulePage>>(`${SCHEDULE_BASE}/list`, params);
  return {
    list: result.data?.list ?? [],
    total: result.data?.total ?? 0,
    current: result.data?.current ?? params.current,
    pageSize: result.data?.pageSize ?? params.pageSize,
  };
}

/** 查询单条调度详情 */
export async function getSchedule(id: string): Promise<ScheduleRecord | null> {
  const result = await post<ScheduleRecord>(`${SCHEDULE_BASE}/${id}`, {});
  return result.data ?? null;
}

/** 新建调度 */
export async function createSchedule(payload: SchedulePayload): Promise<void> {
  await post(SCHEDULE_BASE, payload);
}

/** 更新调度（后端按「非空字段」部分更新） */
export async function updateSchedule(id: string, payload: Partial<SchedulePayload>): Promise<void> {
  await post(`${SCHEDULE_BASE}/${id}/update`, payload);
}

/** 删除调度 */
export async function deleteSchedule(id: string): Promise<void> {
  await post(`${SCHEDULE_BASE}/${id}/delete`, {});
}

/** 立即触发一次（与到点触发共用同一执行入口） */
export async function triggerSchedule(id: string): Promise<void> {
  await post(`${SCHEDULE_BASE}/${id}/trigger`, {});
}

/** 启用/停用调度（status：0=启用 1=停用） */
export async function setScheduleStatus(id: string, status: number): Promise<void> {
  await post(`${SCHEDULE_BASE}/${id}/status`, { status });
}

// ---------------------------------------------------------------------------
// 关联任务（供「关联任务」下拉与列表页任务名映射使用）
// ---------------------------------------------------------------------------

/** 任务列表项的最小视图 */
export interface TaskBrief {
  id: string;
  /** 任务名称（界面展示用） */
  taskName: string;
}

/** 任务列表查询参数 */
export interface TaskBriefParams {
  current: number;
  pageSize: number;
  /** 按任务名称模糊搜索 */
  name?: string;
}

/**
 * 查询 RPA 任务（关联任务数据源）。
 *
 * 分页上限取 200：既够覆盖常规规模的关联任务选择，
 * 又不触发「大分页下拉」反模式（页面 lint 规则对 dropdown loader 的硬约束）。
 */
export async function listTaskBriefs(
  params: TaskBriefParams
): Promise<{ list: TaskBrief[]; total: number }> {
  const result = await post<{ list?: TaskBrief[]; total?: number }>("/rpa/tasks/list", params);
  return {
    list: result.data?.list ?? [],
    total: result.data?.total ?? 0,
  };
}
