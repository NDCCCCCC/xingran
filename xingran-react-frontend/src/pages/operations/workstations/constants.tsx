/**
 * Workstation Constants
 * 工位常量定义
 */

import { Tag } from "antd";
import type { WorkstationNode } from "@/components/shared/FloorPlanEditor.types";
import type { WorkstationOps } from "@/types";
import { WORKSTATION_STATUS_OPTIONS, WORKSTATION_STATUS_TAG_CONFIG } from "@/constants/status";

// 工位状态选项（Phase 69 DICT-03: 共享常量别名引用——工位状态是三态业务簇
// （对齐 models.WorkstationStatus: 0=空闲/1=占用/2=维护），非通用启停 0/1，独立成组）
// 字面量拷贝（浅拷贝，元素共享引用）：react-refresh 对「export const X = 大写标识符」
// 误判为组件导出，ArrayExpression 初始化可消除误判；无就地 mutate、无引用相等比较
export const STATUS_OPTIONS = [...WORKSTATION_STATUS_OPTIONS];

// 工位类型选项（Phase 69 DICT-03: 降级为静态 fallback——编辑弹窗下拉已迁
// useDict("ops_workstation_type")，字典空态/接口异常时回退渲染本数组；导出保留不删除）
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

// 获取工位类型文本（静态映射 = 字典空态/接口异常时的降级）
export function getWorkstationTypeText(type: number): string {
  return TYPE_TEXT_MAP[type] || "-";
}

/**
 * 字典感知的工位类型文本解析。
 *
 * 背景：工位类型是**数据字典**（ops_workstation_type），管理员可在字典管理页
 * 新增类型（如「虚拟工位」）。若只用硬编码的 TYPE_TEXT_MAP（仅 0/1/2 三项），
 * 新增字典项在列表/卡片/平面图会显示为 "-"。
 *
 * 口径：优先按字典 dictValue → dictLabel 解析；字典未命中时降级静态映射；
 * 两者皆无才返回 "-"。字典数据由页面层 useDict("ops_workstation_type") 提供，
 * 本模块不调 hook（保持纯函数，便于测试与复用）。
 */
export function resolveWorkstationTypeText(
  type: number,
  dictItems?: ReadonlyArray<{ dictValue: string; dictLabel: string }>
): string {
  if (dictItems) {
    const hit = dictItems.find((d) => Number(d.dictValue) === type);
    if (hit?.dictLabel) return hit.dictLabel;
  }
  return getWorkstationTypeText(type);
}

/**
 * 字典感知的工位类型标签渲染（与 resolveWorkstationTypeText 同口径）。
 */
export function renderWorkstationTypeTagWithDict(
  type: number,
  dictItems?: ReadonlyArray<{ dictValue: string; dictLabel: string }>
) {
  return <Tag>{resolveWorkstationTypeText(type, dictItems)}</Tag>;
}

// 获取工位状态文本（Phase 69 DICT-03: 改用共享 tag 配置，删除本地重复映射）
export function getWorkstationStatusText(status: number): string {
  return WORKSTATION_STATUS_TAG_CONFIG[status]?.text || "-";
}

// 获取工位状态颜色
export function getWorkstationStatusColor(status: number): string {
  return WORKSTATION_STATUS_TAG_CONFIG[status]?.color || "default";
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
