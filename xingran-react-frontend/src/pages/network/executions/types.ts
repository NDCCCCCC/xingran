/**
 * Config Execution Types
 * 配置执行页面类型定义
 */

import type { NetworkDevice, ConfigTemplate, ConfigExecution, ConfigExecutionDetail } from "@/types";

/** 执行状态 */
export type ExecutionStatus = "pending" | "running" | "success" | "failed";

/** 统计数据 */
export interface ExecutionStatistics {
  total: number;
  pending: number;
  running: number;
  success: number;
  failed: number;
}

/** 模态框状态 */
export interface ModalState {
  executeModalVisible: boolean;
  variableModalVisible: boolean;
  detailDrawerVisible: boolean;
}

/** 执行数据状态 */
export interface ExecutionDataState {
  devices: NetworkDevice[];
  templates: ConfigTemplate[];
  executions: ConfigExecution[];
  executionDetails: ConfigExecutionDetail[];
  currentExecution: ConfigExecution | null;
  selectedTemplate: ConfigTemplate | null;
}
