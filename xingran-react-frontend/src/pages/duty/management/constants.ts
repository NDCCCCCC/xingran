/**
 * Duty Management Constants
 * 值班管理常量定义
 */

/** 调班原因选项 */
export const SWAP_REASON_OPTIONS = [
  { label: "临时有事", value: "临时有事" },
  { label: "身体原因", value: "身体原因" },
  { label: "工作原因", value: "工作原因" },
  { label: "其他", value: "其他" },
] as const;

/** 手动排班备注选项 */
export const MANUAL_REASON_OPTIONS = [
  { label: "临时排班", value: "临时排班" },
  { label: "替班", value: "替班" },
  { label: "补充排班", value: "补充排班" },
  { label: "其他", value: "其他" },
] as const;

/** 节假日名称选项 */
export const HOLIDAY_NAME_OPTIONS = [
  { label: "元旦", value: "元旦" },
  { label: "春节", value: "春节" },
  { label: "清明节", value: "清明节" },
  { label: "劳动节", value: "劳动节" },
  { label: "端午节", value: "端午节" },
  { label: "中秋节", value: "中秋节" },
  { label: "国庆节", value: "国庆节" },
] as const;

/** 节假日类型选项 */
export const HOLIDAY_TYPE_OPTIONS = [
  { label: "法定节假日", value: "legal" },
  { label: "调休工作日", value: "workday" },
  { label: "自定义", value: "custom" },
] as const;

/** 节假日备注选项 */
export const HOLIDAY_REMARK_OPTIONS = [
  { label: "放假", value: "放假" },
  { label: "调休", value: "调休" },
] as const;

/** 批量节假日名称选项 */
export const BATCH_HOLIDAY_NAME_OPTIONS = [
  { label: "春节", value: "春节" },
  { label: "国庆节", value: "国庆节" },
] as const;

/** Excel 导入最大天数 */
export const MAX_BATCH_DAYS = 90;

/** 值班类型选项 */
export const DUTY_TYPE_OPTIONS = [
  { label: "工作日", value: "weekday" },
  { label: "周末", value: "weekend" },
  { label: "节假日", value: "holiday" },
] as const;
