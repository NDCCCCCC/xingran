/**
 * CAD 编辑器颜色主题
 */

// ==================== 颜色主题 ====================

export const CAD_COLOR_THEME = {
  // 背景色
  background: '#fafafa',

  // 网格
  grid: '#e8e8e8',
  gridMajor: '#d0d0d0',

  // 墙体颜色
  wall: {
    default: '#5C6BC0',    // 靛蓝色
    exterior: '#455A64',   // 外墙深灰
    selected: '#1890ff',   // 选中亮蓝
    hover: '#7986CB',      // 悬停
  },

  // 门颜色
  door: {
    default: '#FF7043',    // 橙色
    emergency: '#F5222D',  // 紧急红色
    selected: '#ff4d4f',   // 选中
    hover: '#FF8A65',      // 悬停
  },

  // 工位颜色
  workstation: {
    available: '#52c41a',  // 空闲绿色
    occupied: '#ff4d4f',   // 占用红色
    maintain: '#faad14',   // 维护橙色
    selected: '#1890ff',   // 选中
    hover: '#69c0ff',      // 悬停
  },

  // 辅助元素
  guide: {
    line: '#ffec3d',       // 辅助线黄色
    snap: '#52c41a',       // 吸附点绿色
  },

  // 测量
  measurement: {
    line: '#8c8c8c',       // 测量线灰色
    text: '#262626',       // 测量文字
    background: 'rgba(255, 255, 255, 0.9)',
  },

  // 选择框
  selection: {
    fill: 'rgba(24, 144, 255, 0.1)',
    stroke: '#1890ff',
  },
} as const;

// ==================== 颜色获取函数 ====================

/**
 * 获取墙体颜色
 */
export function getWallColor(
  wall: { type?: string; color?: string },
  selected: boolean = false,
  hovered: boolean = false
): string {
  if (selected) return CAD_COLOR_THEME.wall.selected;
  if (hovered) return CAD_COLOR_THEME.wall.hover;
  if (wall.color) return wall.color;
  if (wall.type === 'exterior') return CAD_COLOR_THEME.wall.exterior;
  return CAD_COLOR_THEME.wall.default;
}

/**
 * 获取门颜色
 */
export function getDoorColor(
  door: { type?: string; color?: string },
  selected: boolean = false,
  hovered: boolean = false
): string {
  if (selected) return CAD_COLOR_THEME.door.selected;
  if (hovered) return CAD_COLOR_THEME.door.hover;
  if (door.color) return door.color;
  if (door.type === 'emergency') return CAD_COLOR_THEME.door.emergency;
  return CAD_COLOR_THEME.door.default;
}

/**
 * 获取工位颜色
 */
export function getWorkstationColor(
  workstation: { status?: number; color?: string },
  selected: boolean = false,
  hovered: boolean = false
): string {
  if (selected) return CAD_COLOR_THEME.workstation.selected;
  if (hovered) return CAD_COLOR_THEME.workstation.hover;
  if (workstation.color) return workstation.color;

  // 0 = 空闲, 1 = 占用, 2 = 维护
  switch (workstation.status) {
    case 0:
      return CAD_COLOR_THEME.workstation.available;
    case 1:
      return CAD_COLOR_THEME.workstation.occupied;
    case 2:
      return CAD_COLOR_THEME.workstation.maintain;
    default:
      return CAD_COLOR_THEME.workstation.available;
  }
}

// ==================== 默认颜色 ====================

export const DEFAULT_WALL_COLOR = CAD_COLOR_THEME.wall.default;
export const DEFAULT_DOOR_COLOR = CAD_COLOR_THEME.door.default;
export const DEFAULT_WORKSTATION_COLOR = CAD_COLOR_THEME.workstation.available;

// ==================== 类型导出 ====================

export type CadColorTheme = typeof CAD_COLOR_THEME;
