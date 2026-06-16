/**
 * 楼层管理页面工具函数
 */

import type { WorkstationOps } from "@/types";
import type { WorkstationNode } from "@/components/shared/FloorPlanEditor.types";
import { WORKSTATION_LAYOUT } from "./constants";

/**
 * 处理工位数据：为没有位置的工位自动分配位置
 */
export function processWorkstations(workstations: WorkstationOps[]): WorkstationNode[] {
  const { DEFAULT_WIDTH, DEFAULT_DEPTH, GAP, ITEMS_PER_ROW, START_X, START_Y } = WORKSTATION_LAYOUT;

  // 分离有位置和无位置的工位
  const withPosition: WorkstationNode[] = [];
  const withoutPosition: WorkstationOps[] = [];

  workstations.forEach((ws) => {
    if (ws.positionX !== undefined && ws.positionY !== undefined) {
      // 有位置的工位：使用数据库中的实际值（直接使用，单位已经是像素）
      withPosition.push({
        id: ws.id,
        code: ws.name,
        name: ws.name,
        x: ws.positionX,
        y: ws.positionY,
        width: ws.width || DEFAULT_WIDTH,
        height: ws.depth || DEFAULT_DEPTH,
        rotation: ws.rotation || 0,
        type: ws.deskType || 0,
        status: ws.status,
      });
    } else {
      withoutPosition.push(ws);
    }
  });

  // 为没有位置的工位自动分配位置
  withoutPosition.forEach((ws, index) => {
    const row = Math.floor(index / ITEMS_PER_ROW);
    const col = index % ITEMS_PER_ROW;

    const wsWidth = ws.width || DEFAULT_WIDTH;
    const wsDepth = ws.depth || DEFAULT_DEPTH;

    withPosition.push({
      id: ws.id,
      code: ws.name,
      name: ws.name,
      x: START_X + col * (wsWidth + GAP),
      y: START_Y + row * (wsDepth + GAP),
      width: wsWidth,
      height: wsDepth,
      rotation: ws.rotation || 0,
      type: ws.deskType || 0,
      status: ws.status,
    });
  });

  return withPosition;
}

/**
 * 解析JSON字段
 * 当解析失败时返回原值（类型可能不匹配，需要调用方处理）
 */
export function parseJsonField<T>(value: string | T): T {
  if (typeof value === "string") {
    try {
      return JSON.parse(value) as T;
    } catch {
      // JSON解析失败，返回原字符串作为T类型
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      return value as any;
    }
  }
  return value;
}

/**
 * 序列化JSON字段
 */
export function stringifyJsonField(value: unknown): string {
  if (typeof value === "string") {
    return value;
  }
  return JSON.stringify(value);
}

/**
 * 判断是否为新创建的元素
 */
export function isNewElement(id: string, prefix: string): boolean {
  return id.startsWith(prefix);
}
