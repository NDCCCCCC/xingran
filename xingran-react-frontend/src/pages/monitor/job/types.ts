/**
 * Job Types
 * 定时任务管理类型定义
 */

/** 定时任务信息 */
export interface JobInfo {
  id: string;
  jobName: string;
  jobGroup: string;
  invokeTarget: string;
  cronExpression: string;
  misfirePolicy: number;
  concurrent: boolean;
  status: number;
  createdAt: string;
  updatedAt: string;
  createdBy: string;
  updatedBy: string;
  remark?: string;
  nextRunTime?: string;
  prevRunTime?: string;
}

/** 任务执行日志 */
export interface JobLog {
  id: string;
  jobName: string;
  jobGroup: string;
  invokeTarget: string;
  jobMessage: string;
  status: number;
  exceptionInfo?: string;
  startTime?: string;
  endTime?: string;
  duration: number;
  createdAt: string;
}

/** 分页数据 */
export interface PageData {
  list: JobInfo[];
  total: number;
  current: number;
  pageSize: number;
  data?: {
    list: JobInfo[];
    total: number;
  };
}

/** 搜索表单状态 */
export interface SearchFormState {
  jobName: string;
  jobGroup: string;
  status: number | undefined;
}

/** 任务状态枚举 */
export enum JobStatus {
  Normal = 0,
  Paused = 1,
}

/** 错过执行策略枚举 */
export enum MisfirePolicy {
  ExecuteImmediately = 1,
  ExecuteOnce = 2,
  Discard = 3,
}
