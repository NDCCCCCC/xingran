// 楼宇空间可视化常量定义

// ============ 地图相关常量 ============

/** 百度地图 AK */
export const BAIDU_MAP_AK = import.meta.env.VITE_BAIDU_MAP_AK || "";

/** 地图配置 */
export const MAP_CONFIG = {
  /** 湖北省中心点 */
  CENTER: [114.305393, 30.593099] as [number, number],
  /** 默认缩放级别（显示整个湖北省） */
  DEFAULT_ZOOM: 8 as number,
  /** 最小缩放级别 */
  MIN_ZOOM: 8,
  /** 最大缩放级别 */
  MAX_ZOOM: 18,
  /** 3D 倾斜角度 */
  DEFAULT_TILT: 60,
  /** 默认旋转角度 */
  DEFAULT_HEADING: 0,
  /** 地图样式 ID */
  STYLE_ID: "c6cd6c1c7b622236b02f5fcc53e7f18",
} as const;

/** 湖北省简化边界坐标（降级方案） */
export const HUBEI_BOUNDARY: [number, number][] = [
  [109.5, 33.5],
  [111.5, 33.5],
  [113.5, 33.2],
  [115.5, 33.0],
  [116.5, 32.5],
  [116.2, 31.8],
  [116.1, 31.0],
  [115.8, 30.2],
  [115.5, 29.5],
  [115.2, 29.0],
  [114.8, 28.5],
  [114.0, 28.2],
  [113.0, 28.0],
  [112.0, 28.2],
  [111.0, 28.5],
  [110.0, 29.0],
  [109.5, 29.8],
  [109.2, 30.5],
  [109.0, 31.2],
  [109.0, 32.0],
  [109.2, 32.5],
  [109.5, 33.5],
];

/** 聚类像素阈值 */
export const CLUSTER_PIXEL_THRESHOLD = 40;

/** 聚类标记最小尺寸 */
export const CLUSTER_MARKER_MIN_SIZE = 48;

/** 聚类标记最大增量尺寸 */
export const CLUSTER_MARKER_MAX_INCREMENT = 24;

// ============ 3D 场景相关常量 ============

/** 工位尺寸配置 */
export const WORKSTATION_DIMENSIONS = {
  /** 桌面宽度 */
  DESK_WIDTH: 1.2,
  /** 桌面深度 */
  DESK_DEPTH: 0.8,
  /** 桌面高度 */
  DESK_HEIGHT: 0.04,
  /** 桌腿高度 */
  LEG_HEIGHT: 0.75,
  /** 桌腿粗细 */
  LEG_THICKNESS: 0.06,
  /** 椅子座位宽度 */
  CHAIR_WIDTH: 0.42,
  /** 椅子座位高度 */
  CHAIR_SEAT_HEIGHT: 0.04,
  /** 椅子靠背高度 */
  CHAIR_BACK_HEIGHT: 0.35,
  /** 椅子靠背厚度 */
  CHAIR_BACK_THICKNESS: 0.04,
  /** 椅子座位离地高度 */
  CHAIR_SEAT_Y: 0.45,
  /** 椅子总高度 */
  CHAIR_TOTAL_HEIGHT: 0.7,
  /** 椅子腿高度 */
  CHAIR_LEG_HEIGHT: 0.44,
  /** 椅子腿粗细 */
  CHAIR_LEG_THICKNESS: 0.025,
  /** 扶手高度 */
  ARMREST_HEIGHT: 0.12,
  /** 扶手粗细 */
  ARMREST_THICKNESS: 0.04,
  /** 扶手水平偏移 */
  ARMREST_OFFSET: 0.23,
} as const;

/** 显示器尺寸配置 */
export const MONITOR_DIMENSIONS = {
  /** 屏幕宽度 */
  WIDTH: 0.45,
  /** 屏幕高度 */
  HEIGHT: 0.3,
  /** 屏幕厚度 */
  THICKNESS: 0.02,
  /** 边框宽度增量 */
  BORDER_INCREMENT: 0.02,
  /** 边框高度增量 */
  BORDER_HEIGHT_INCREMENT: 0.02,
  /** 边框厚度增量 */
  BORDER_THICKNESS_INCREMENT: 0.005,
  /** 支架宽度 */
  STAND_WIDTH: 0.12,
  /** 支架高度 */
  STAND_HEIGHT: 0.24,
  /** 支架深度 */
  STAND_DEPTH: 0.08,
  /** 支架底部 Y 坐标 */
  STAND_BOTTOM_Y: 0.12,
  /** 屏幕 Y 坐标 */
  SCREEN_Y: 0.28,
} as const;

/** 键盘尺寸配置 */
export const KEYBOARD_DIMENSIONS = {
  /** 键盘宽度 */
  WIDTH: 0.32,
  /** 键盘深度 */
  DEPTH: 0.14,
  /** 键盘高度 */
  HEIGHT: 0.015,
  /** 按键高度 */
  KEY_HEIGHT: 0.008,
  /** 按键宽度缩减 */
  KEY_WIDTH_REDUCTION: 0.02,
  /** 按键深度缩减 */
  KEY_DEPTH_REDUCTION: 0.02,
} as const;

/** 鼠标垫尺寸配置 */
export const MOUSEPAD_DIMENSIONS = {
  /** 宽度 */
  WIDTH: 0.18,
  /** 深度 */
  DEPTH: 0.22,
  /** 高度 */
  HEIGHT: 0.01,
} as const;

/** 工位自动排列配置 */
export const WORKSTATION_LAYOUT = {
  /** 每行工位数量 */
  GRID_SIZE: 8,
  /** 单元格大小（工位间距） */
  CELL_SIZE: 1.5,
  /** 预设位置转换系数 */
  POSITION_SCALE: 10,
  /** 预设位置偏移量 */
  POSITION_OFFSET: 50,
} as const;

/** 3D 场景相机配置 */
export const CAMERA_CONFIG = {
  /** 默认位置 */
  DEFAULT_POSITION: [0, 15, 15] as [number, number, number],
  /** 视野角度 */
  FOV: 50,
  /** 最小距离 */
  MIN_DISTANCE: 5,
  /** 最大距离 */
  MAX_DISTANCE: 40,
  /** 最大极角（限制俯视角度） */
  MAX_POLAR_ANGLE: Math.PI / 2.5,
} as const;

/** 3D 场景地面配置 */
export const FLOOR_CONFIG = {
  /** 地面尺寸 */
  SIZE: 200,
  /** 地面 Y 坐标 */
  Y: -0.01,
  /** 地面颜色 */
  COLOR: "#ffffff",
} as const;

// ============ 颜色常量 ============

/** 工位状态颜色 */
export const WORKSTATION_STATUS_COLORS = {
  /** 空闲 - 深绿色 */
  AVAILABLE: 0x388e3c,
  /** 占用 - 深红色 */
  OCCUPIED: 0xd32f2f,
  /** 维护 - 深橙色 */
  MAINTENANCE: 0xf57c00,
  /** 灵活工位 - 深紫色 */
  FLEXIBLE: 0x7b1fa2,
  /** 管理工位 - 深青色 */
  MANAGERIAL: 0x13c2c2,
} as const;

/** 材质颜色 */
export const MATERIAL_COLORS = {
  /** 桌腿颜色 */
  DESK_LEG: 0x2d2d2d,
  /** 桌面颜色 */
  DESK_TOP: 0xf5f5f5,
  /** 椅子座位颜色 */
  CHAIR_SEAT: 0x3d3d3d,
  /** 扶手颜色 */
  ARMREST: 0x2d2d2d,
  /** 椅子腿颜色 */
  CHAIR_LEG: 0x1a1a1a,
  /** 显示器支架颜色 */
  MONITOR_STAND: 0x333333,
  /** 显示器屏幕颜色 */
  MONITOR_SCREEN: 0x1a1a1a,
  /** 显示器边框颜色 */
  MONITOR_BORDER: 0x2d2d2d,
  /** 键盘颜色 */
  KEYBOARD: 0x2d2d2d,
  /** 键盘按键颜色 */
  KEYBOARD_KEY: 0x1a1a1a,
  /** 鼠标垫颜色 */
  MOUSEPAD: 0x4a4a4a,
} as const;

// ============ UI 样式常量 ============

/** 标记颜色 */
export const MARKER_COLORS = {
  /** 停用楼宇主色 */
  STOPPED_MAIN: "#9e9e9e",
  /** 停用楼宇边框色 */
  STOPPED_BORDER: "#757575",
  /** 停用楼宇阴影色 */
  STOPPED_SHADOW: "rgba(0,0,0,0.2)",
  /** 正常楼宇主色 */
  NORMAL_MAIN: "#ff6b35",
  /** 正常楼宇边框色 */
  NORMAL_BORDER: "#ff4d4f",
  /** 正常楼宇阴影色 */
  NORMAL_SHADOW: "rgba(255, 107, 53, 0.5)",
  /** 聚类标记颜色 */
  CLUSTER: "#ff4d4f",
  /** 聚类标记背景色 */
  CLUSTER_BG: "rgba(255,77,79,0.1)",
  /** 聚类标记阴影色 */
  CLUSTER_SHADOW: "rgba(255, 77, 79, 0.5)",
} as const;

/** 状态文本颜色 */
export const STATUS_TEXT_COLORS = {
  /** 停用状态文本色 */
  STOPPED: "#9e9e9e",
  /** 停用状态背景色 */
  STOPPED_BG: "#f5f5f5",
  /** 正常状态文本色 */
  NORMAL: "#ff6b35",
  /** 正常状态背景色 */
  NORMAL_BG: "#fff2e8",
} as const;

// ============ 动画配置 ============

/** 动画时长 */
export const ANIMATION_DURATION = {
  /** 默认过渡动画 */
  DEFAULT: 1500,
  /** 3D 视角切换 */
  TOGGLE_3D: 1200,
} as const;

// ============ 缓动函数 ============

/** 缓动函数集合 */
export const EASING_FUNCTIONS = {
  /** easeInOutCubic - 先慢后快再慢 */
  easeInOutCubic: (t: number): number => {
    return t < 0.5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2;
  },
  /** easeInOutQuad - 平滑的加速减速 */
  easeInOutQuad: (t: number): number => {
    return t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;
  },
  /** easeOutCubic - 快速开始，缓慢结束 */
  easeOutCubic: (t: number): number => {
    return 1 - Math.pow(1 - t, 3);
  },
} as const;

// ============ 文本常量 ============

/** 状态文本 */
export const STATUS_TEXT = {
  /** 正常 */
  NORMAL: "正常",
  /** 停用 */
  STOPPED: "停用",
  /** 空闲 */
  AVAILABLE: "空闲",
  /** 占用 */
  OCCUPIED: "占用",
  /** 维护 */
  MAINTENANCE: "维护",
  /** 固定工位 */
  FIXED: "固定",
  /** 灵活工位 */
  FLEXIBLE: "灵活",
  /** 管理工位 */
  MANAGERIAL: "管理",
} as const;
