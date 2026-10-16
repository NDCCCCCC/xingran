/**
 * Device Discovery Types
 * 设备发现页面类型定义
 */

/** IP 范围 */
export interface IPRange {
  startIP: string;
  endIP: string;
}

/** 发现状态 */
export type DiscoveryStatus = "pending" | "running" | "completed" | "failed";

/** 统计数据 */
export interface DiscoveryStatistics {
  total: number;
  pending: number;
  running: number;
  completed: number;
  failed: number;
  totalDevices: number;
}

/** 模态框状态 */
export interface ModalState {
  modalVisible: boolean;
  resultModalVisible: boolean;
}
