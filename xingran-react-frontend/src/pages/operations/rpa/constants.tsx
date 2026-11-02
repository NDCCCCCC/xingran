/**
 * RPA 常量定义
 */

import { Tag } from "antd";

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
export const TASK_STATUS_OPTIONS = [
  { label: "草稿", value: 0 },
  { label: "启用", value: 1 },
  { label: "禁用", value: 2 },
];

// 任务状态文本映射
const TASK_STATUS_TEXT_MAP: Record<number, string> = {
  0: "草稿",
  1: "启用",
  2: "禁用",
};

// 任务状态颜色映射
const TASK_STATUS_COLOR_MAP: Record<number, string> = {
  0: "default",
  1: "success",
  2: "error",
};

// ==================== 执行状态 ====================
export const EXECUTION_STATUS_OPTIONS = [
  { label: "等待中", value: "pending" },
  { label: "运行中", value: "running" },
  { label: "成功", value: "success" },
  { label: "失败", value: "failed" },
  { label: "已取消", value: "cancelled" },
];

// 执行状态文本映射
const EXECUTION_STATUS_TEXT_MAP: Record<string, string> = {
  pending: "等待中",
  running: "运行中",
  success: "成功",
  failed: "失败",
  cancelled: "已取消",
};

// 执行状态颜色映射
const EXECUTION_STATUS_COLOR_MAP: Record<string, string> = {
  pending: "default",
  running: "processing",
  success: "success",
  failed: "error",
  cancelled: "default",
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
export const ACTION_TYPE_OPTIONS = [
  { label: "导航", value: "navigate" },
  { label: "点击", value: "click" },
  { label: "输入", value: "fill" },
  { label: "等待", value: "wait" },
  { label: "选择", value: "select" },
  { label: "上传", value: "upload" },
  { label: "提取", value: "extract" },
  { label: "断言", value: "assert" },
  { label: "脚本", value: "script" },
];

// 脚本动作文本映射
export const ACTION_TYPE_TEXT_MAP: Record<string, string> = {
  navigate: "导航",
  click: "点击",
  fill: "输入",
  wait: "等待",
  select: "选择",
  upload: "上传",
  extract: "提取",
  assert: "断言",
  script: "自定义脚本",
};
