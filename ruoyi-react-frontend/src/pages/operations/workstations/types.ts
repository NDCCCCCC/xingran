/**
 * Workstation Types
 * 工位类型定义
 */

import type { WorkstationOps } from "@/types";

// 视图模式
export type ViewMode = "table" | "card" | "floorplan";

// 统计数据
export interface WorkstationStatistics {
  total: number;
  available: number;
  occupied: number;
  maintain: number;
}

// 楼层选项
export interface FloorOption {
  id: string;
  code: string;
  name: string;
}

// 用户选项
export interface UserOption {
  id: string;
  username: string;
  nickname?: string;
}

// 部门树节点
export interface DeptTreeNode {
  title: string;
  value: string;
  key: string;
  children?: DeptTreeNode[];
}
