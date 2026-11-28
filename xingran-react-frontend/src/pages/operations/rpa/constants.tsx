/**
 * RPA 常量定义
 */

import { Tag } from "antd";
import { RPA_ACTION_TYPES, type ActionType } from "@/types/rpa";

// ==================== 触发器类型 ====================
export const TRIGGER_TYPE_OPTIONS = [
  { label: "手动触发", value: "manual" },
  { label: "定时触发", value: "schedule" },
  { label: "事件触发", value: "event" },
  { label: "Webhook", value: "webhook" },
];

// 触发器类型文本映射
const TRIGGER_TYPE_TEXT_MAP: Record<string, string> = {
  manual: "手动触发",
  schedule: "定时触发",
  event: "事件触发",
  webhook: "Webhook",
};

// 触发器类型颜色映射
const TRIGGER_TYPE_COLOR_MAP: Record<string, string> = {
  manual: "default",
  schedule: "blue",
  event: "orange",
  webhook: "purple",
};

// ==================== 任务状态 ====================
// CONTRACT-02：任务状态是「任务态」而非执行态 —— 与后端 models/rpa/task.go
// 的 TaskStatus 枚举严格同构：0=启用(TaskStatusEnabled) / 1=停用(TaskStatusDisabled)。
// 历史前端这里是 0草稿/1启用/2禁用，既多出一个后端不存在的 2，又把 0 的含义
// 从「启用」错标成「草稿」，导致筛选与列表标签双双对不上真实数据。
export const TASK_STATUS_OPTIONS = [
  { label: "启用", value: 0 },
  { label: "停用", value: 1 },
];

// 任务状态文本映射
const TASK_STATUS_TEXT_MAP: Record<number, string> = {
  0: "启用",
  1: "停用",
};

// 任务状态颜色映射
const TASK_STATUS_COLOR_MAP: Record<number, string> = {
  0: "success",
  1: "default",
};

// ==================== 执行状态 ====================
// CONTRACT-02：必须覆盖后端 RPAExecutionStatus 全部 6 个取值
// （pending/running/success/failed/cancelled/timeout）。历史缺 timeout ——
// 超时的执行记录在按状态筛选时无从选中，列表标签也回落成 "-"。
export const EXECUTION_STATUS_OPTIONS = [
  { label: "等待中", value: "pending" },
  { label: "运行中", value: "running" },
  { label: "成功", value: "success" },
  { label: "失败", value: "failed" },
  { label: "已取消", value: "cancelled" },
  { label: "已超时", value: "timeout" },
];

// 执行状态文本映射
const EXECUTION_STATUS_TEXT_MAP: Record<string, string> = {
  pending: "等待中",
  running: "运行中",
  success: "成功",
  failed: "失败",
  cancelled: "已取消",
  timeout: "已超时",
};

// 执行状态颜色映射
const EXECUTION_STATUS_COLOR_MAP: Record<string, string> = {
  pending: "default",
  running: "processing",
  success: "success",
  failed: "error",
  cancelled: "default",
  timeout: "warning",
};

// ==================== Worker 状态 ====================
export const WORKER_STATUS_OPTIONS = [
  { label: "在线", value: "online" },
  { label: "离线", value: "offline" },
  { label: "忙碌", value: "busy" },
  { label: "错误", value: "error" },
];

// Worker 状态文本映射
const WORKER_STATUS_TEXT_MAP: Record<string, string> = {
  online: "在线",
  offline: "离线",
  busy: "忙碌",
  error: "错误",
};

// Worker 状态颜色映射
const WORKER_STATUS_COLOR_MAP: Record<string, string> = {
  online: "success",
  offline: "default",
  busy: "processing",
  error: "error",
};

// ==================== 辅助函数 ====================

// 获取触发器类型文本
export function getTriggerTypeText(type: string): string {
  return TRIGGER_TYPE_TEXT_MAP[type] || "-";
}

// 获取触发器类型颜色
export function getTriggerTypeColor(type: string): string {
  return TRIGGER_TYPE_COLOR_MAP[type] || "default";
}

// 渲染触发器类型标签
export function renderTriggerTypeTag(type: string) {
  return <Tag color={getTriggerTypeColor(type)}>{getTriggerTypeText(type)}</Tag>;
}

// 获取任务状态文本
export function getTaskStatusText(status: number): string {
  return TASK_STATUS_TEXT_MAP[status] || "-";
}

// 获取任务状态颜色
export function getTaskStatusColor(status: number): string {
  return TASK_STATUS_COLOR_MAP[status] || "default";
}

// 渲染任务状态标签
export function renderTaskStatusTag(status: number) {
  return <Tag color={getTaskStatusColor(status)}>{getTaskStatusText(status)}</Tag>;
}

// 获取执行状态文本
export function getExecutionStatusText(status: string): string {
  return EXECUTION_STATUS_TEXT_MAP[status] || "-";
}

// 获取执行状态颜色
export function getExecutionStatusColor(status: string): string {
  return EXECUTION_STATUS_COLOR_MAP[status] || "default";
}

// 渲染执行状态标签
export function renderExecutionStatusTag(status: string) {
  return <Tag color={getExecutionStatusColor(status)}>{getExecutionStatusText(status)}</Tag>;
}

// 获取 Worker 状态文本
export function getWorkerStatusText(status: string): string {
  return WORKER_STATUS_TEXT_MAP[status] || "-";
}

// 获取 Worker 状态颜色
export function getWorkerStatusColor(status: string): string {
  return WORKER_STATUS_COLOR_MAP[status] || "default";
}

// 渲染 Worker 状态标签
export function renderWorkerStatusTag(status: string) {
  return <Tag color={getWorkerStatusColor(status)}>{getWorkerStatusText(status)}</Tag>;
}

// ==================== 脚本动作类型 ====================

/**
 * 脚本动作文本映射（REC-03：类型改为 Record<ActionType, string>）
 *
 * 编译期强制覆盖共享枚举 RPA_ACTION_TYPES 的全部 26 个动作 —— 少一个就直接
 * tsc 报错。中文名不得含 ASCII 字母（面向普通员工，错误信息不暴露英文）。
 */
export const ACTION_TYPE_TEXT_MAP: Record<ActionType, string> = {
  navigate: "导航",
  click: "点击",
  fill: "输入",
  select: "选择",
  wait: "等待",
  screenshot: "截图",
  extract: "提取",
  scroll: "滚动",
  upload: "上传",
  download: "下载",
  evaluate: "执行页面脚本",
  waitFor: "等待元素出现",
  close: "关闭页面",
  loop: "循环执行",
  pause: "暂停待人工",
  condition: "条件判断",
  autologin: "自动登录",
  assert: "断言",
  hover: "悬停",
  doubleClick: "双击",
  rightClick: "右键点击",
  drag: "拖拽",
  keyboard: "按键",
  switchTab: "切换页签",
  iframe: "切换内嵌页面",
  script: "自定义脚本",
};

/**
 * 脚本动作类型下拉选项（REC-03：从共享枚举派生）
 *
 * 历史上这里是手写的 9 项，而 worker 能执行 26 种动作 —— 下拉里根本选不到
 * extract 之外的大多数动作，属于典型的双源漂移。现在直接遍历 RPA_ACTION_TYPES
 * 生成，下拉项永远与共享枚举同源。
 */
export const ACTION_TYPE_OPTIONS: Array<{ label: string; value: ActionType }> =
  RPA_ACTION_TYPES.map((type) => ({
    label: ACTION_TYPE_TEXT_MAP[type],
    value: type,
  }));
