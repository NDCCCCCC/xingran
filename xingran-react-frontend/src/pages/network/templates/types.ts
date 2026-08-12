/**
 * Network Template Types
 * 网络模板类型定义
 */

import type { ConfigTemplate } from "@/types";

// 统计数据
export interface TemplateStatistics {
  total: number;
  system: number;
  custom: number;
  init: number;
}

// 模态框状态
export interface TemplateModalState {
  editModalVisible: boolean;
  previewVisible: boolean;
  variablesModalVisible: boolean;
  editingTemplate: ConfigTemplate | null;
  selectedRowKeys: React.Key[];
  previewContent: string;
  templateVariables: Record<string, unknown>;
}

// 选项类型
export interface SelectOption {
  label: string;
  value: string;
}
