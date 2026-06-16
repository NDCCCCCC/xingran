/**
 * Workstation Constants
 * 工位常量定义
 */

import { Tag } from "antd";
import type { WorkstationNode } from "@/components/shared/FloorPlanEditor.types";
import type { WorkstationOps } from "@/types";

// 工位状态选项
export const STATUS_OPTIONS = [
  { label: "空闲", value: 0 },
  { label: "占用", value: 1 },
  { label: "维护", value: 2 },
];

// 工位类型选项
export const TYPE_OPTIONS = [
  { label: "固定工位", value: 0 },
  { label: "灵活工位", value: 1 },
  { label: "管理工位", value: 2 },
];

// 工位类型文本映射
const TYPE_TEXT_MAP: Record<number, string> = {
  0: "固定工位",
  1: "灵活工位",
  2: "管理工位",
};

// 工位状态文本映射
const STATUS_TEXT_MAP: Record<number, string> = {
  0: "空闲",
  1: "占用",
  2: "维护",
};

// 工位状态颜色映射
const STATUS_COLOR_MAP: Record<number, string> = {
  0: "success",
  1: "error",
  2: "warning",
};

// 获取工位类型文本
export function getWorkstationTypeText(type: number): string {
  return TYPE_TEXT_MAP[type] || "-";
}

// 获取工位状态文本
export function getWorkstationStatusText(status: number): string {
  return STATUS_TEXT_MAP[status] || "-";
}

// 获取工位状态颜色
export function getWorkstationStatusColor(status: number): string {
  return STATUS_COLOR_MAP[status] || "default";
}

// 渲染工位类型标签
export function renderWorkstationTypeTag(type: number) {
  return <Tag>{getWorkstationTypeText(type)}</Tag>;
}

// 渲染工位状态标签
export function renderWorkstationStatusTag(status: number) {
  return <Tag color={getWorkstationStatusColor(status)}>{getWorkstationStatusText(status)}</Tag>;
}

// 转换为平面图节点格式
export function toWorkstationNode(w: WorkstationOps): WorkstationNode {
  return {
    id: w.id,
    code: w.name,
    name: w.name,
    x: w.positionX || 0,
    y: w.positionY || 0,
    width: w.width || 160,
    height: w.depth || 70,
    status: w.status,
    type: w.type,
    rotation: w.rotation || 0,
  };
}
