/**
 * Config Execution Constants
 * 配置执行页面常量定义
 */

import type { ExecutionStatus } from "./types";
import { Tag } from "antd";
import { ClockCircleOutlined, CheckCircleOutlined, CloseCircleOutlined } from "@ant-design/icons";

/** 执行状态选项 */
export const STATUS_OPTIONS = [
  { label: "待执行", value: "pending" },
  { label: "执行中", value: "running" },
  { label: "成功", value: "success" },
  { label: "失败", value: "failed" },
] as const;

/** 执行状态配置 */
export const STATUS_CONFIG: Record<
  ExecutionStatus,
  { color: string; icon: React.ReactNode; text: string }
> = {
  pending: { color: "default", icon: <ClockCircleOutlined />, text: "待执行" },
  running: { color: "processing", icon: <ClockCircleOutlined />, text: "执行中" },
  success: { color: "success", icon: <CheckCircleOutlined />, text: "成功" },
  failed: { color: "error", icon: <CloseCircleOutlined />, text: "失败" },
};

/** 渲染执行状态标签 */
export function renderStatusTag(status: ExecutionStatus) {
  const config = STATUS_CONFIG[status] || STATUS_CONFIG.pending;
  return (
    <Tag color={config.color} icon={config.icon}>
      {config.text}
    </Tag>
  );
}

/** 渲染简化执行状态标签（无图标） */
export function renderSimpleStatusTag(status: ExecutionStatus) {
  const config = STATUS_CONFIG[status] || STATUS_CONFIG.pending;
  return <Tag color={config.color}>{config.text}</Tag>;
}
