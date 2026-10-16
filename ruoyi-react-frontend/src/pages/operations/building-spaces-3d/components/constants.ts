/**
 * 3D 楼宇视图组件共享常量
 */

// 地图配置
export const MAP_CONFIG = {
  // 湖北省中心点
  CENTER: [114.305393, 30.593099] as const,
  // 默认缩放级别
  DEFAULT_ZOOM: 8,
  // 最小缩放级别
  MIN_ZOOM: 8,
  // 最大缩放级别（普通版本）
  MAX_ZOOM: 10,
  // WebGL 版本最大缩放
  MAX_ZOOM_GL: 18,
  // 3D 倾斜角度
  DEFAULT_TILT: 60,
  // 默认旋转角度
  DEFAULT_HEADING: 0,
} as const;

// 湖北省边界坐标（简化版）
export const HUBEI_BOUNDARY = [
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
] as const;

// 楼宇标记颜色配置
export const MARKER_COLORS = {
  // 正常状态
  NORMAL: {
    MAIN: "#ff6b35",
    BORDER: "#ba3630",
    SHADOW: "rgba(255, 107, 53, 0.5)",
  },
  // 停用状态
  STOPPED: {
    MAIN: "#9e9e9e",
    BORDER: "#757575",
    SHADOW: "rgba(0, 0, 0, 0.2)",
  },
  // 聚类标记
  CLUSTER: {
    MAIN: "#ba3630",
    SHADOW: "rgba(186, 54, 48, 0.5)",
  },
  // 边界
  BOUNDARY: "#337ab0",
} as const;

// 3D 楼宇视图状态色（业务专属色，暂不归入 design-system）
// 后续如需主题切换可考虑迁移到 design-system/tokens/colors.ts
export const THREE_D_STATUS_COLORS = {
  ANOMALY: { MAIN: "#ff6b35", BORDER: "#ba3630" },
  INACTIVE: { MAIN: "#9e9e9e", BORDER: "#757575" },
  ABNORMAL: { MAIN: "#ba3630", BORDER: "#ba3630" },
  BOUNDARY: { MAIN: "#337ab0", BORDER: "#096dd9" },
  MAINTENANCE: { MAIN: "#f57c00", BORDER: "#d46b08" },
} as const;

// 3D 楼层颜色配置
export const FLOOR_3D_COLORS = {
  STOPPED: 0xd32f2f, // 停用 - 深红色
  NO_WORKSTATION: 0xf57c00, // 无工位 - 深橙色
  HIGH_OCCUPANCY: 0x388e3c, // 高占用率 - 深绿色
  NORMAL: 0x1976d2, // 正常 - 深蓝色
} as const;

// 3D 工位颜色配置
export const WORKSTATION_3D_COLORS = {
  OCCUPIED: 0xd32f2f, // 占用 - 深红色
  MAINTENANCE: 0xf57c00, // 维护 - 深橙色
  FLEXIBLE: 0x7b1fa2, // 灵活 - 深紫色
  MANAGEMENT: 0x13c2c2, // 管理 - 深青色
  AVAILABLE: 0x388e3c, // 空闲 - 深绿色
} as const;

// 3D 场景尺寸配置
export const SCENE_DIMENSIONS = {
  FLOOR: {
    HEIGHT: 0.4,
    SIZE: 4.5,
    SPACING: 1.2,
  },
  WORKSTATION: {
    DESK_WIDTH: 1.2,
    DESK_DEPTH: 0.8,
    DESK_HEIGHT: 0.04,
    LEG_HEIGHT: 0.75,
  },
} as const;

// 聚类配置
export const CLUSTER_CONFIG = {
  PIXEL_THRESHOLD: 40, // 聚类阈值（像素）
  MIN_SIZE: 48, // 最小聚类图标大小
  SIZE_MULTIPLIER: 2, // 大小增长系数
  MAX_SIZE_ADD: 24, // 最大额外大小
} as const;

// 样式配置
export const STYLE_CONFIG = {
  OVERLAY: {
    BACKGROUND: "rgba(255, 255, 255, 0.95)",
    BORDER: "1px solid #e8e8e8",
    BORDER_RADIUS: "8px",
    BOX_SHADOW: "0 2px 8px rgba(0, 0, 0, 0.1)",
    PADDING: "12px 16px",
  },
  BUTTON: {
    GRADIENT: "linear-gradient(135deg, #337ab0, #096dd9)",
    BOX_SHADOW: "0 2px 4px rgba(51, 122, 176, 0.3)",
  },
} as const;

// 状态文本映射
export const STATUS_TEXT = {
  BUILDING: {
    NORMAL: "正常",
    STOPPED: "已停用",
  },
  FLOOR: {
    NORMAL: "正常",
    STOPPED: "停用",
  },
  WORKSTATION: {
    AVAILABLE: "空闲",
    OCCUPIED: "占用",
    MAINTENANCE: "维护",
    FIXED: "固定",
    FLEXIBLE: "灵活",
    MANAGEMENT: "管理",
  },
} as const;

// 状态颜色映射（Ant Design Tag）
export const STATUS_COLOR = {
  BUILDING: {
    NORMAL: "success",
    STOPPED: "default",
  } as const,
  FLOOR: {
    NORMAL: "success",
    STOPPED: "default",
  } as const,
  WORKSTATION_STATUS: {
    AVAILABLE: "success",
    OCCUPIED: "error",
    MAINTENANCE: "warning",
  } as const,
  WORKSTATION_TYPE: {
    FIXED: "blue",
    FLEXIBLE: "purple",
    MANAGEMENT: "cyan",
  } as const,
} as const;

// 缩放级别显示的楼宇层级
export const getVisibleBuildingLevels = (zoom: number): "all" | "level1" => {
  return zoom >= 10 ? "all" : "level1";
};
