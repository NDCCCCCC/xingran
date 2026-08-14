import { post } from "./api";
import type { BaseResponse, PageResponse } from "@/types";

// ==================== 类型定义 ====================

export interface DutyPool {
  id: string;
  poolName: string;
  deptId?: string;
  department?: { id: string; deptName: string };
  description?: string;
  status: number; // 0=启用 1=停用
  dailyCount: number;
  members?: DutyPoolMember[];
  createdAt: string;
  createdBy: string;
  updatedAt?: string;
  updatedBy?: string;
}

export interface DutyPoolMember {
  id: string;
  poolId: string;
  userId: string;
  user?: {
    id: string;
    username: string;
    nickname: string;
    deptName?: string;
  };
  memberOrder: number;
  createdAt: string;
}

export interface DutySchedule {
  id: string;
  scheduleDate: string;
  poolId: string;
  pool?: { id: string; poolName: string };
  userId: string;
  user?: {
    id: string;
    username: string;
    nickname: string;
    deptName?: string;
  };
  dutyType: "weekday" | "weekend" | "holiday";
  status: number; // 0=正常 1=已调换 2=已取消
  isManual: boolean;
  swapFromDate?: string;
  swapReason?: string;
  createdAt: string;
  createdBy: string;
  updatedAt?: string;
  updatedBy?: string;
}

export interface Holiday {
  id: string;
  holidayDate: string;
  holidayName: string;
  isOffday: boolean; // true=休息日 false=调休工作日
  holidayType: "legal" | "workday" | "custom";
  year: number;
  remark?: string;
  createdAt: string;
  createdBy: string;
  updatedAt?: string;
  updatedBy?: string;
}

export interface DutyConfig {
  id: string;
  reminderEnabled: boolean;
  reminderTime: string; // HH:mm格式
  reminderChannels: string; // websocket,email,sms
  beforeReminderMinutes?: number;
  createdAt: string;
  updatedAt?: string;
}

// ==================== 请求参数类型 ====================

export interface DutyPoolListRequest {
  current?: number;
  pageSize?: number;
  poolName?: string;
  status?: number;
  deptId?: string;
  /** 服务端排序字段（对应后端 dutyPoolAllowedSortFields 白名单 key） */
  orderByColumn?: string;
  /** 是否升序 */
  isAsc?: boolean;
}

export interface DutyPoolCreateRequest {
  poolName: string;
  deptId?: string;
  description?: string;
  dailyCount: number;
  memberIds: string[];
}

export interface DutyPoolUpdateRequest {
  poolName?: string;
  deptId?: string;
  description?: string;
  status?: number;
  dailyCount?: number;
  memberIds?: string[];
}

export interface DutyScheduleListRequest {
  current?: number;
  pageSize?: number;
  poolId?: string;
  userId?: string;
  startDate?: string;
  endDate?: string;
  dutyType?: string;
  /** 过期状态：0=未过期，1=已过期，不传=全部 */
  expired?: number;
  /** 服务端排序字段（对应后端 dutyScheduleAllowedSortFields 白名单 key） */
  orderByColumn?: string;
  /** 是否升序 */
  isAsc?: boolean;
}

export interface GenerateScheduleRequest {
  poolId: string;
  startDate: string;
  endDate: string;
  dutyType: "weekday" | "weekend" | "holiday";
  clearExists?: boolean;
}

export interface SwapDutyRequest {
  fromScheduleId: string;
  toScheduleId: string;
  reason?: string;
}

export interface ManualDutyRequest {
  poolId: string;
  dutyDate: string;
  dutyType: "weekday" | "weekend" | "holiday";
  userIds: string[];
  reason?: string;
}

// ==================== API函数 ====================

// ==================== 值班池管理 ====================

export function getDutyPoolList(
  params: DutyPoolListRequest
): Promise<BaseResponse<PageResponse<DutyPool>>> {
  return post("/duty/pools/list", params);
}

/** 值班池统计（总数 / 启用 / 停用 / 成员总数） */
export interface DutyPoolStatistics {
  total: number;
  enabled: number;
  disabled: number;
  totalMembers: number;
}

/**
 * 获取值班池统计（后端 COUNT 聚合，不受分页上限影响）
 */
export function getDutyPoolStatistics(): Promise<BaseResponse<DutyPoolStatistics>> {
  return post("/duty/pools/statistics", {});
}

export function createDutyPool(data: DutyPoolCreateRequest): Promise<BaseResponse<DutyPool>> {
  return post("/duty/pools", data);
}

export function getDutyPool(id: string): Promise<BaseResponse<DutyPool>> {
  return post(`/duty/pools/${id}`, {});
}

export function updateDutyPool(
  id: string,
  data: DutyPoolUpdateRequest
): Promise<BaseResponse<{ message: string }>> {
  return post(`/duty/pools/${id}/update`, data);
}

export function deleteDutyPool(id: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/duty/pools/${id}/delete`);
}

// ==================== 排班管理 ====================

export function getDutyScheduleList(
  params: DutyScheduleListRequest
): Promise<BaseResponse<PageResponse<DutySchedule>>> {
  return post("/duty/schedules/list", params);
}

export function generateSchedule(
  data: GenerateScheduleRequest
): Promise<BaseResponse<{ message: string; count: number }>> {
  return post("/duty/schedules/generate", data);
}

export function getTodayDuty(): Promise<
  BaseResponse<{
    date: string;
    members: Array<{ userId: string; username: string; nickname: string }>;
  }>
> {
  return post("/duty/schedules/today", {});
}

export function swapDuty(data: SwapDutyRequest): Promise<BaseResponse<{ message: string }>> {
  return post("/duty/schedules/swap", data);
}

export function manualDuty(data: ManualDutyRequest): Promise<BaseResponse<{ message: string }>> {
  return post("/duty/schedules/manual", data);
}

export function deleteDutySchedule(id: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/duty/schedules/${id}/delete`);
}

export function batchDeleteDutySchedules(
  ids: string[]
): Promise<BaseResponse<{ message: string; count: number }>> {
  return post("/duty/schedules/batch-delete", { ids });
}

export interface MonthlyDutyMember {
  scheduleId: string;
  poolId: string;
  poolName: string;
  userId: string;
  username: string;
  phone: string;
  dutyType: string;
}

export function getMonthlyDutySchedule(
  year: number,
  month: number
): Promise<BaseResponse<Record<string, MonthlyDutyMember[]>>> {
  return post("/duty/schedules/monthly", { year, month });
}

// ==================== 节假日管理 ====================

export function getHolidayList(year: number): Promise<BaseResponse<Holiday[]>> {
  return post("/duty/holidays/list", { year });
}

export function createHoliday(
  data: Omit<Holiday, "id" | "createdAt" | "createdBy">
): Promise<BaseResponse<Holiday>> {
  return post("/duty/holidays", data);
}

export function updateHoliday(
  id: string,
  data: Partial<Omit<Holiday, "id" | "createdAt" | "createdBy">>
): Promise<BaseResponse<{ message: string }>> {
  return post(`/duty/holidays/${id}/update`, data);
}

export function deleteHoliday(id: string): Promise<BaseResponse<{ message: string }>> {
  return post(`/duty/holidays/${id}/delete`);
}

export function batchCreateHolidays(
  holidays: Omit<Holiday, "id" | "createdAt" | "createdBy">[]
): Promise<BaseResponse<{ message: string; count: number }>> {
  return post("/duty/holidays/batch", { holidays });
}

export function getHolidayYears(): Promise<BaseResponse<number[]>> {
  return post("/duty/holidays/years", {});
}

// ==================== 值班配置管理 ====================

export function getDutyConfig(): Promise<BaseResponse<DutyConfig>> {
  return post("/duty/config", {});
}

export function updateDutyConfig(
  data: Partial<DutyConfig>
): Promise<BaseResponse<{ message: string }>> {
  return post("/duty/config/update", data);
}

// ==================== 用户和部门（用于下拉选择）====================

export interface SimpleUser {
  id: string;
  username: string;
  nickname: string;
  deptId?: string;
  deptName?: string;
  status: number;
}

export interface SimpleDept {
  id: string;
  deptName: string;
  parentId?: string;
  children?: SimpleDept[];
}

// 获取用户列表（用于下拉选择）
// recursiveDeptId: 后端按该部门+所有子部门递归查询用户（基于 sys_dept.ancestors）。
// 用于值班池/工位等"选部门后加载该部门成员"场景——避免预加载全量用户被 MaxPageSize 钳制。
export function getUserList(params?: {
  current?: number;
  pageSize?: number;
  deptId?: string;
  recursiveDeptId?: string;
  status?: number;
}): Promise<BaseResponse<PageResponse<SimpleUser>>> {
  return post("/system/users/list", {
    current: 1,
    pageSize: 1000,
    ...params,
  });
}

// 获取部门列表（用于下拉选择）
export function getDeptList(): Promise<BaseResponse<SimpleDept[]>> {
  return post("/system/departments/list");
}

// 获取部门树（用于下拉选择）
export function getDeptTree(): Promise<BaseResponse<SimpleDept[]>> {
  return post("/system/departments/tree");
}

// ==================== 我的值班 ====================

export interface MyDutyStats {
  isOnDutyToday: boolean;
  todayDutyRecords?: Array<{
    scheduleId: string;
    poolId: string;
    poolName: string;
    userId: string;
    dutyType: string;
  }>;
  thisMonthCount: number;
  totalCount: number;
  nextDutyDate?: string;
  nextDutyPoolName?: string;
}

export function getMyDutyStats(): Promise<BaseResponse<MyDutyStats>> {
  return post("/duty/my-duty/stats", {});
}
