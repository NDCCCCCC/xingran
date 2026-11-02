/**
 * Workstation Types
 * 工位类型定义
 */

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
  /**
   * 是否为外部机构(0/1)。来源是 sys_dept.is_external_org。
   * 必须在 useWorkstationData.buildTreeData 中透传,否则 filterExternalOrgDepts
   * 会拿到 undefined 而误判,导致 orgTreeData 为空。
   */
  isExternalOrg?: number;
  children?: DeptTreeNode[];
}
