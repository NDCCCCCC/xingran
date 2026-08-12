/**
 * Log Constants
 * 日志管理常量定义
 */

import { BusinessType, LogStatus } from "./types";

/** 业务类型选项 */
export const BUSINESS_TYPE_OPTIONS: Array<{ label: string; value: BusinessType }> = [
  { label: "其它", value: BusinessType.Other },
  { label: "新增", value: BusinessType.Create },
  { label: "修改", value: BusinessType.Update },
  { label: "删除", value: BusinessType.Delete },
  { label: "授权", value: BusinessType.Grant },
  { label: "导出", value: BusinessType.Export },
  { label: "导入", value: BusinessType.Import },
  { label: "强退", value: BusinessType.ForceLogout },
  { label: "生成代码", value: BusinessType.GenerateCode },
  { label: "清空数据", value: BusinessType.ClearData },
];

/** 日志状态选项 */
export const LOG_STATUS_OPTIONS: Array<{ label: string; value: LogStatus }> = [
  { label: "正常", value: LogStatus.Success },
  { label: "异常", value: LogStatus.Failure },
];

/** 登录状态选项 */
export const LOGIN_STATUS_OPTIONS: Array<{ label: string; value: LogStatus }> = [
  { label: "成功", value: LogStatus.Success },
  { label: "失败", value: LogStatus.Failure },
];

/** 默认搜索表单值 */
export const DEFAULT_OPER_SEARCH_FORM = {
  title: "",
  businessType: undefined,
  status: undefined,
  operName: "",
  timeRange: null as any,
};

export const DEFAULT_LOGIN_SEARCH_FORM = {
  userName: "",
  ipAddr: "",
  status: undefined,
  timeRange: null as any,
};
