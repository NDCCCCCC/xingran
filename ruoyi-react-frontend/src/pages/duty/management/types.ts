/**
 * Duty Management Types
 * 值班管理类型定义
 */

import type dayjs from "dayjs";
import type { Holiday } from "@/lib/dutyApi";

/** 排班搜索参数 */
export interface ScheduleSearchParams {
  poolId?: string;
  userId?: string;
  dutyType?: string;
  dateRange?: [dayjs.Dayjs, dayjs.Dayjs];
  /** 过期状态：0=未过期，1=已过期 */
  expired?: number;
}

/** 生成排班值 */
export interface GenerateScheduleValues {
  poolId: string;
  startDate: string;
  endDate: string;
  dutyType: string;
  clearExists?: boolean;
}

/** 值班配置值 */
export interface DutyConfigValues {
  reminderEnabled: boolean;
  reminderTime: dayjs.Dayjs;
  reminderChannels: string[];
  beforeReminderMinutes?: number;
}

/** 节假日 Excel 行数据 */
export interface HolidayExcelRow {
  "日期(YYYY-MM-DD)"?: string;
  "日期"?: string;
  "名称"?: string;
  "节假日名称"?: string;
  "类型(legal/workday/custom)"?: string;
  "类型"?: string;
  "是否休息(true/false)"?: boolean;
  "备注"?: string;
}

/** 导入选项 */
export interface ImportOptions {
  file: File;
  onProgress?: (event: { percent: number }) => void;
  onSuccess?: (response?: unknown) => void;
  onError?: (error: Error) => void;
}

/** 节假日创建数据 */
export type HolidayCreateData = Omit<Holiday, "id" | "createdAt" | "updatedAt" | "createdBy">;

/** 批量节假日表单值 */
export interface BatchHolidayFormValues {
  dateRange: [dayjs.Dayjs, dayjs.Dayjs];
  holidayName: string;
  holidayType: "legal" | "custom";
  isOffday: boolean;
  remark?: string;
}
