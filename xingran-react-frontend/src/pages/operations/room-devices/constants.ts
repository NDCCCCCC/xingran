import type { CSSProperties } from "react";
import type { RoomDevice } from "@/types";

export type { RoomDevice };

export type ViewMode = "table" | "card";

export interface RoomOption {
  id: string;
  name: string;
  buildingId: string;
  orgId: string;
}

export interface Statistics {
  total: number;
  normal: number;
  fault: number;
  scrapped: number;
}

/**
 * 卡片视图图标动作的禁用态样式（Phase 86 W2，与 buildings 页同口径）。
 * antd `Card.actions` 只接受 ReactNode → 未就绪时用「不响应指针 + 降透明度」
 * 表达禁用态，并配合 `title={decision.hint}` 提示原因。
 */
export const cardActionStyle = (disabled: boolean): CSSProperties => ({
  opacity: disabled ? 0.45 : 1,
  pointerEvents: disabled ? "none" : "auto",
  cursor: disabled ? "not-allowed" : "pointer",
});

// 模块级静态映射（Phase m76-js-misc 6.3）
export const ROOM_DEVICE_STATUS_MAP: Record<number, string> = { 0: "正常", 1: "故障", 2: "报废" };

// 搜索区选项（值沿用后端 status 口径 0/1/2）
export const ROOM_DEVICE_STATUS_OPTIONS = [
  { label: "正常", value: 0 },
  { label: "故障", value: 1 },
  { label: "报废", value: 2 },
];

// 设备类型选项（来自字典 device_type，由 useDict 注入）
export type DeviceTypeOption = { label: string; value: string };

export const getStatusText = (status: number): string => {
  return ROOM_DEVICE_STATUS_MAP[status] || "未知";
};

// 未命中字典时显示原值（存量数据兜底）
export const getDeviceTypeText = (type: string, options: DeviceTypeOption[] = []): string => {
  const option = options.find((item) => item.value === type);
  return option ? option.label : type;
};
