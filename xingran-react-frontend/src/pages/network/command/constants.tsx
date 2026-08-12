/**
 * Network Command Constants
 * 网络命令常量定义
 */

import { Tag } from "antd";
import {
  ClockCircleOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from "@ant-design/icons";

// 执行状态选项
export const STATUS_OPTIONS = [
  { label: "待执行", value: "pending" },
  { label: "执行中", value: "running" },
  { label: "成功", value: "success" },
  { label: "失败", value: "failed" },
];

// 执行状态配置
export const STATUS_CONFIG: Record<
  string,
  { color: string; icon: React.ReactNode; text: string }
> = {
  pending: { color: "default", icon: <ClockCircleOutlined />, text: "待执行" },
  running: { color: "processing", icon: <ClockCircleOutlined />, text: "执行中" },
  success: { color: "success", icon: <CheckCircleOutlined />, text: "成功" },
  failed: { color: "error", icon: <CloseCircleOutlined />, text: "失败" },
};

// 简单状态配置（无图标）
export const SIMPLE_STATUS_CONFIG: Record<string, { color: string; text: string }> = {
  pending: { color: "default", text: "待执行" },
  running: { color: "processing", text: "执行中" },
  success: { color: "success", text: "成功" },
  failed: { color: "error", text: "失败" },
};

// 渲染执行状态标签（带图标）
export function renderExecutionStatusTag(status: string) {
  const config = STATUS_CONFIG[status] || STATUS_CONFIG.pending;
  return <Tag color={config.color} icon={config.icon}>{config.text}</Tag>;
}

// 渲染简单状态标签（无图标）
export function renderSimpleStatusTag(status: string) {
  const config = SIMPLE_STATUS_CONFIG[status] || SIMPLE_STATUS_CONFIG.pending;
  return <Tag color={config.color}>{config.text}</Tag>;
}
