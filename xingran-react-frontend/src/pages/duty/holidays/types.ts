/**
 * Holiday Types
 * 节假日页面类型定义
 */

import type { Dayjs } from "dayjs";
import type { Holiday } from "@/lib/dutyApi";

/** 节假日类型 */
export type HolidayType = "legal" | "workday" | "custom";

/** 批量添加行数据 */
export interface BatchHolidayRow {
  holidayDate: Dayjs;
  holidayName: string;
  isOffday: boolean;
  holidayType: HolidayType;
  year: number;
  remark?: string;
}

/** 模态框状态 */
export interface ModalState {
  modalVisible: boolean;
  batchModalVisible: boolean;
  editingRecord: Holiday | null;
}

/** 批量操作状态 */
export interface BatchState {
  batchHolidays: BatchHolidayRow[];
}

/** Excel 导入选项 */
export interface ExcelImportOptions {
  file: File;
  onSuccess?: (data: unknown) => void;
  onError?: (error: Error) => void;
}

/** Excel 行数据（解析后） */
export interface ExcelHolidayRow {
  holidayDate: string;
  holidayName: string;
  isOffday: boolean;
  holidayType: HolidayType;
  year: number;
  remark?: string;
}
