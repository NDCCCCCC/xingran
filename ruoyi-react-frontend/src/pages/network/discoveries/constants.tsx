/**
 * Device Discovery Constants
 * 设备发现页面常量定义
 */

import { Tag } from "antd";
import type { DiscoveryStatus } from "./types";

/** 发现类型选项 */
export const DISCOVERY_TYPE_OPTIONS = [
  { label: "SNMP扫描", value: "snmp" },
  { label: "网络扫描", value: "scan" },
] as const;

/** 发现状态选项 */
export const STATUS_OPTIONS = [
  { label: "待执行", value: "pending" },
  { label: "扫描中", value: "running" },
  { label: "已完成", value: "completed" },
  { label: "失败", value: "failed" },
] as const;

/** 发现状态配置 */
export const STATUS_CONFIG: Record<DiscoveryStatus, { color: string; text: string }> = {
  pending: { color: "default", text: "待执行" },
  running: { color: "processing", text: "扫描中" },
  completed: { color: "success", text: "已完成" },
  failed: { color: "error", text: "失败" },
};

/** 渲染状态标签 */
export function renderStatusTag(status: DiscoveryStatus) {
  const config = STATUS_CONFIG[status] || STATUS_CONFIG.pending;
  return <Tag color={config.color}>{config.text}</Tag>;
}
