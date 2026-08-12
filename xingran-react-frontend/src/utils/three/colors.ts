// 颜色方案
import { brandColors } from "@/design-system/tokens/colors";

/**
 * 工位状态颜色（同步系统主题颜色）
 */
export const WORKSTATION_STATUS_COLORS = {
  0: {
    main: brandColors.success,      // 绿色 - 空闲
    label: "空闲",
    bg: "#f6ffed",
    border: "#b7eb8f",
  },
  1: {
    main: brandColors.error,        // 红色 - 占用/禁用
    label: "占用",
    bg: "#fff1f0",
    border: "#ffccc7",
  },
  2: {
    main: brandColors.warning,      // 橙色 - 维护
    label: "维护",
    bg: "#fffbe6",
    border: "#ffe58f",
  },
} as const;

/**
 * 工位类型颜色（同步系统主题颜色）
 */
export const WORKSTATION_TYPE_COLORS = {
  0: {
    main: brandColors.primary,      // 蓝色 - 固定
    label: "固定",
    bg: "#e6f7ff",
    border: "#91d5ff",
  },
  1: {
    main: brandColors.secondary,    // 紫色 - 灵活
    label: "灵活",
    bg: "#f9f0ff",
    border: "#d3adf7",
  },
  2: {
    main: brandColors.info,         // 青色 - 管理
    label: "管理",
    bg: "#e6fffb",
    border: "#87e8de",
  },
} as const;

/**
 * 地图标记颜色
 */
export const MAP_MARKER_COLORS = {
  city: {
    main: "#1890ff",       // 蓝色 - 城市标记
    bg: "#e6f7ff",
    border: "#91d5ff",
  },
  building: {
    normal: "#52c41a",     // 绿色 - 正常楼宇
    stopped: "#9e9e9e",    // 灰色 - 停用
    bg: "#f6ffed",
    border: "#b7eb8f",
  },
  selected: {
    main: "#ff4d4f",       // 红色 - 选中
    bg: "#fff1f0",
    border: "#ffccc7",
  },
} as const;

/**
 * 楼宇 3D 模型颜色
 */
export const BUILDING_3D_COLORS = {
  normal: 0x667eea,        // 正常
  stopped: 0x9e9e9e,       // 停用
  hover: 0x764ba2,         // 悬停
  selected: 0xff6b6b,      // 选中
} as const;

/**
 * 获取工位状态颜色
 */
export function getWorkstationStatusColor(status: number) {
  return WORKSTATION_STATUS_COLORS[status as keyof typeof WORKSTATION_STATUS_COLORS] || WORKSTATION_STATUS_COLORS[0];
}

/**
 * 获取工位类型颜色
 */
export function getWorkstationTypeColor(type: number) {
  return WORKSTATION_TYPE_COLORS[type as keyof typeof WORKSTATION_TYPE_COLORS] || WORKSTATION_TYPE_COLORS[0];
}

/**
 * 获取地图标记颜色
 */
export function getMapMarkerColor(type: "city" | "building", status?: number) {
  if (type === "city") {
    return MAP_MARKER_COLORS.city;
  }

  if (status === 1) {
    return { ...MAP_MARKER_COLORS.building, main: MAP_MARKER_COLORS.building.stopped };
  }

  return MAP_MARKER_COLORS.building;
}
