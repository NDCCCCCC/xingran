/**
 * CAD 编辑器颜色主题
 */

// ==================== 常量定义 ====================

/** 工位状态: 0 = 空闲, 1 = 占用, 2 = 维护 */
export const WORKSTATION_STATUS = {
  AVAILABLE: 0,
  OCCUPIED: 1,
  MAINTAIN: 2,
} as const;

/** 默认吸附距离 */
export const DEFAULT_SNAP_DISTANCE = 10;

/** 默认网格大小 */
export const DEFAULT_GRID_SIZE = 20;

// ==================== 颜色主题 ====================

/** CAD 编辑器颜色主题 */
export const CAD_COLOR_THEME = {
  /** 背景色 */
  background: "#fafafa",

  /** 网格颜色 */
  grid: "#e8e8e8",
  gridMajor: "#d0d0d0",

  /** 墙体颜色 */
  wall: {
    default: "#5C6BC0",
    exterior: "#455A64",
    selected: "#337ab0",
    hover: "#7986CB",
  },

  /** 门颜色 */
  door: {
    default: "#FF7043",
    emergency: "#F5222D",
    selected: "#ba3630",
    hover: "#FF8A65",
  },

  /** 工位颜色 */
  workstation: {
    available: "#2d8949",
    occupied: "#ba3630",
    maintain: "#b07a20",
    selected: "#337ab0",
    hover: "#69c0ff",
  },

  /** 辅助元素颜色 */
  guide: {
    line: "#ffec3d",
    snap: "#2d8949",
  },

  /** 测量元素颜色 */
  measurement: {
    line: "#707068",
    text: "#262626",
    background: "rgba(255, 255, 255, 0.9)",
  },

  /** 选择框颜色 */
  selection: {
    fill: "rgba(51, 122, 176, 0.1)",
    stroke: "#337ab0",
  },
} as const;

// ==================== 颜色获取函数 ====================

/**
 * 获取墙体颜色
 */
export function getWallColor(
  wall: { type?: string; color?: string },
  selected = false,
  hovered = false
): string {
  if (selected) return CAD_COLOR_THEME.wall.selected;
  if (hovered) return CAD_COLOR_THEME.wall.hover;
  if (wall.color) return wall.color;
  if (wall.type === "exterior") return CAD_COLOR_THEME.wall.exterior;
  return CAD_COLOR_THEME.wall.default;
}

/**
 * 获取门颜色
 */
export function getDoorColor(
  door: { type?: string; color?: string },
  selected = false,
  hovered = false
): string {
  if (selected) return CAD_COLOR_THEME.door.selected;
  if (hovered) return CAD_COLOR_THEME.door.hover;
  if (door.color) return door.color;
  if (door.type === "emergency") return CAD_COLOR_THEME.door.emergency;
  return CAD_COLOR_THEME.door.default;
}

/**
 * 获取工位颜色
 */
export function getWorkstationColor(
  workstation: { status?: number; color?: string },
  selected = false,
  hovered = false
): string {
  if (selected) return CAD_COLOR_THEME.workstation.selected;
  if (hovered) return CAD_COLOR_THEME.workstation.hover;
  if (workstation.color) return workstation.color;

  switch (workstation.status) {
    case WORKSTATION_STATUS.AVAILABLE:
      return CAD_COLOR_THEME.workstation.available;
    case WORKSTATION_STATUS.OCCUPIED:
      return CAD_COLOR_THEME.workstation.occupied;
    case WORKSTATION_STATUS.MAINTAIN:
      return CAD_COLOR_THEME.workstation.maintain;
    default:
      return CAD_COLOR_THEME.workstation.available;
  }
}

// ==================== 默认颜色 ====================

export const DEFAULT_WALL_COLOR = CAD_COLOR_THEME.wall.default;
export const DEFAULT_DOOR_COLOR = CAD_COLOR_THEME.door.default;
export const DEFAULT_WORKSTATION_COLOR = CAD_COLOR_THEME.workstation.available;

// ==================== 类型定义 ====================

/** CAD 颜色主题类型 */
export type CadColorTheme = typeof CAD_COLOR_THEME;

// Phase 83 CR-01 trial: harmless appended comment line (cad-editor whitelist is excluded).
