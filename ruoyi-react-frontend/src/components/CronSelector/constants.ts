export type CronFieldType = "second" | "minute" | "hour" | "day" | "month" | "week";
export type PeriodType = "every" | "specific" | "range" | "cycle";

export interface CronFieldConfig {
  type: CronFieldType;
  periodType: PeriodType;
  specific?: number[];
  rangeStart?: number;
  rangeEnd?: number;
  cycleStart?: number;
  cycleInterval?: number;
}

export interface CronConfig {
  second: CronFieldConfig;
  minute: CronFieldConfig;
  hour: CronFieldConfig;
  day: CronFieldConfig;
  month: CronFieldConfig;
  week: CronFieldConfig;
}

export interface CronPreset {
  label: string;
  value: string;
  description: string;
}

export interface FieldRange {
  min: number;
  max: number;
}

export const FIELD_RANGES: Record<string, FieldRange> = {
  second: { min: 0, max: 59 },
  minute: { min: 0, max: 59 },
  hour: { min: 0, max: 23 },
  day: { min: 1, max: 31 },
  month: { min: 1, max: 12 },
  week: { min: 1, max: 7 }
};

export const WEEK_DAY_NAMES = ["周日", "周一", "周二", "周三", "周四", "周五", "周六"];
export const MONTH_NAMES = ["1月", "2月", "3月", "4月", "5月", "6月", "7月", "8月", "9月", "10月", "11月", "12月"];

export const CRON_PRESETS: CronPreset[] = [
  { label: "每分钟", value: "0 * * * * ?", description: "每分钟的第0秒执行" },
  { label: "每小时", value: "0 0 * * * ?", description: "每小时的第0分0秒执行" },
  { label: "每天零点", value: "0 0 0 * * ?", description: "每天0点0分执行" },
  { label: "每天早上9点", value: "0 0 9 * * ?", description: "每天上午9点执行" },
  { label: "工作日9点", value: "0 0 9 ? * MON-FRI", description: "周一到周五上午9点执行" },
  { label: "每周一零点", value: "0 0 0 ? * MON", description: "每周一凌晨执行" },
  { label: "每月1号零点", value: "0 0 0 1 * ?", description: "每月1号凌晨执行" },
  { label: "每5分钟", value: "0 */5 * * * ?", description: "每5分钟执行一次" },
  { label: "每10分钟", value: "0 */10 * * * ?", description: "每10分钟执行一次" },
  { label: "每30分钟", value: "0 */30 * * * ?", description: "每30分钟执行一次" },
  { label: "每2小时", value: "0 0 */2 * * ?", description: "每2小时执行一次" },
  { label: "每天8/12/18点", value: "0 0 8,12,18 * * ?", description: "每天8点、12点、18点执行" },
];

export const FIELD_LABELS: Record<string, string> = {
  second: "秒",
  minute: "分",
  hour: "时",
  day: "日",
  month: "月",
  week: "周"
};

export const DEFAULT_CRON_EXPRESSION = "0 0 9 * * ?";
