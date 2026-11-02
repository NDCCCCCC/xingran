/**
 * 工位管理相关类型（旧版，用于兼容）
 *
 * @deprecated 请使用 operations.ts 中的 WorkstationOps 类型
 */

import type { Status } from "./base";

/**
 * 工位类型
 * @deprecated 请使用 WorkstationOpsType
 */
export type WorkstationType = 0 | 1 | 2;

/**
 * 工位状态
 * @deprecated 请使用 WorkstationOpsStatus
 */
export type WorkstationStatus = Status;

/**
 * 工位
 * @deprecated 请使用 WorkstationOps
 */
export interface Workstation {
  id: string;
  workstationCode: string;
  workstationName: string;
  deptId?: string;
  deptName?: string;
  location?: string;
  floor?: string;
  workstationType: WorkstationType;
  status: WorkstationStatus;
  capacity: number;
  description?: string;
  userId?: string;
  userName?: string;
  createdAt: string;
  updatedAt: string;
}
