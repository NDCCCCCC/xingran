/**
 * Log Types
 * 日志管理类型定义
 */

/** 操作日志 */
export interface OperLog {
  id: string;
  title: string;
  businessType: number;
  method: string;
  requestMethod: string;
  operName: string;
  nickname?: string; // 用户昵称（可选，后端可能未提供）
  deptName: string;
  operUrl: string;
  operIp: string;
  operLocation: string;
  operParam: string;
  operTime: string;
  status: number;
  errorMessage: string;
}

/** 登录日志 */
export interface LoginLog {
  id: string;
  userName: string;
  nickname?: string; // 用户昵称（可选，后端可能未提供）
  ipAddr: string;
  loginLocation: string;
  browser: string;
  os: string;
  status: number;
  message: string;
  loginTime: string;
}

/** 业务类型 */
export enum BusinessType {
  Other = 0,
  Create = 1,
  Update = 2,
  Delete = 3,
  Grant = 4,
  Export = 5,
  Import = 6,
  ForceLogout = 7,
  GenerateCode = 8,
  ClearData = 9,
}

/** 日志状态 */
export enum LogStatus {
  Success = 0,
  Failure = 1,
}

/** 搜索表单状态 */
export interface SearchFormState {
  title?: string;
  businessType?: number;
  status?: number;
  operName?: string;
  userName?: string;
  ipAddr?: string;
  timeRange: any;
}
