/**
 * Department Types
 * 部门页面类型定义
 */

// 重新导出 Department 类型
export type { Department } from "@/types";

/** 部门用户 */
export interface DeptUser {
  id: string;
  username: string;
  nickname?: string;
  phone?: string;
  email?: string;
}

/** 统计数据 */
export interface DeptStatistics {
  total: number;
  topLevel: number;
  subLevel: number;
}

/** 父级选项 */
export interface ParentOption {
  title: string;
  value: string;
  key: string;
  children?: ParentOption[];
}
