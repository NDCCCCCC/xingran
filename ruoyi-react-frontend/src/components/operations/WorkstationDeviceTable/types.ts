/**
 * 工位设备关联组件类型定义
 */

import type { WorkstationDevice, DeviceSource, DeviceFormData } from "@/types";

// 导出类型，方便外部使用
export type { WorkstationDevice, DeviceSource, DeviceFormData };

// 组件 Props 接口
export interface WorkstationDeviceTableProps {
  workstationId: string;
  expandable?: boolean;
  onDeviceChange?: () => void;
}

// 设备来源枚举和标签
export const DEVICE_SOURCE_LABELS: Record<DeviceSource, string> = {
  ad: "域控",
  asset: "资产",
  manual: "手动",
} as const;
