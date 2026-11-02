/**
 * 工位设备关联组件类型定义
 */

import type { WorkstationDevice, DeviceSource, DeviceFormData } from "@/types";
// 复用 @/types/operations 中已包含全部 DeviceSource 键(含 physical)的标签表,
// 避免本地重复定义遗漏导致 Record<DeviceSource, string> 不完整。
export { DEVICE_SOURCE_LABELS } from "@/types/operations";

// 导出类型，方便外部使用
export type { WorkstationDevice, DeviceSource, DeviceFormData };

// 组件 Props 接口
export interface WorkstationDeviceTableProps {
  workstationId: string;
  expandable?: boolean;
  onDeviceChange?: () => void;
}
