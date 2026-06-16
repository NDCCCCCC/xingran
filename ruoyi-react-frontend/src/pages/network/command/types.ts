/**
 * Network Command Types
 * 网络命令类型定义
 */

import type { ConfigExecution, ConfigExecutionDetail } from "@/types";

// 统计数据
export interface CommandStatistics {
  total: number;
  pending: number;
  running: number;
  success: number;
  failed: number;
}

// 模态框状态
export interface CommandModalState {
  dispatchModalVisible: boolean;
  detailDrawerVisible: boolean;
  selectedRowKeys: string[];
  currentExecution: ConfigExecution | null;
  executionDetails: ConfigExecutionDetail[];
}
