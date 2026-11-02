/**
 * Job Constants
 * 定时任务管理常量定义
 */

import { JobStatus, MisfirePolicy } from "./types";

/** 任务状态选项 */
export const STATUS_OPTIONS = [
  { label: "正常", value: JobStatus.Normal },
  { label: "暂停", value: JobStatus.Paused },
];

/** 错过执行策略选项 */
export const MISFIRE_POLICY_OPTIONS = [
  { label: "立即执行", value: MisfirePolicy.ExecuteImmediately },
  { label: "执行一次", value: MisfirePolicy.ExecuteOnce },
  { label: "放弃执行", value: MisfirePolicy.Discard },
];

/** 默认表单值 */
export const DEFAULT_FORM_VALUES = {
  misfirePolicy: MisfirePolicy.ExecuteImmediately,
  concurrent: false,
  status: JobStatus.Normal,
};

/** 默认搜索表单值 */
export const DEFAULT_SEARCH_FORM = {
  jobName: "",
  jobGroup: "",
  status: undefined as number | undefined,
};
