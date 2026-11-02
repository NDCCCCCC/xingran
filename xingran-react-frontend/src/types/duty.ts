/**
 * 值班管理相关类型
 */

/**
 * 值班类型
 */
export type DutyType = "weekday" | "weekend" | "holiday";

/**
 * 今日值班记录
 */
export interface TodayDutyRecord {
  poolName: string;
  dutyType: DutyType;
}

/**
 * 我的值班统计
 */
export interface MyDutyStats {
  isOnDutyToday: boolean;
  thisMonthCount: number;
  totalCount: number;
  nextDutyDate?: string;
  nextDutyPoolName?: string;
  todayDutyRecords?: TodayDutyRecord[];
}
