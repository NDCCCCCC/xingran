/**
 * 统一配置类型定义
 * Unified Configuration Type Definitions
 *
 * 这是系统所有配置的类型中心，确保类型安全
 */

// ============= 主题配置 =============

/**
 * 主题颜色模式
 */
export type ColorMode = "light" | "dark";

/**
 * 主题配置（v1.22 Phase 65 · D-01 收窄版：仅明暗模式）
 * 旧持久化数据中的 style / customColors 多余键运行时静默忽略。
 */
export interface ThemeConfiguration {
  /** 颜色模式：light / dark */
  mode: ColorMode;
}

// ============= 布局配置 =============

/**
 * 布局类型
 */
export type LayoutType = "classic" | "hybrid" | "innovative";

/**
 * 密度模式
 */
export type DensityMode = "compact" | "comfortable" | "spacious";

/**
 * 布局配置
 */
export interface LayoutConfiguration {
  /** 布局类型 */
  type: LayoutType;

  /** 侧边栏配置 */
  sidebar: {
    /** 是否折叠 */
    collapsed: boolean;
    /** 展开宽度（像素） */
    width: number;
    /** 折叠宽度（像素） */
    collapsedWidth: number;
  };

  /** 密度模式 */
  density: DensityMode;
}

// ============= 数据配置 =============

/**
 * 数据配置
 */
export interface DataConfiguration {
  /** 全局默认分页大小 */
  defaultPageSize: number;

  /** 每页可选项 */
  pageSizeOptions: number[];
}

// ============= 用户偏好设置（主类型）============

/**
 * 用户偏好设置 - 权威数据源
 *
 * 版本历史：
 * - v1: 初始版本（扁平结构：theme, language, pageSize, sidebarCollapsed）
 * - v2: 嵌套结构（theme.mode+style, layout.*, data.*）
 */
export interface UserPreferences {
  /** 版本号，用于数据迁移 */
  version: number;

  /** 主题配置 */
  theme: ThemeConfiguration;

  /** 布局配置 */
  layout: LayoutConfiguration;

  /** 数据配置 */
  data: DataConfiguration;

  /** 语言（预留，暂未实现完整i18n） */
  language?: "zh-CN" | "en-US";
}

// ============= 后端API类型 =============

/**
 * 后端API返回的用户设置格式
 */
export interface BackendUserPreferences {
  // 主题
  theme: string;

  /** @deprecated v1.22 Phase 65 多主题移除；后端契约保留，前端不再消费 */
  themeStyle?: string;

  // 布局
  layoutType?: string;
  layoutDensity?: string;
  sidebarWidth?: number;
  sidebarCollapsedWidth?: number;
  sidebarCollapsed: boolean;

  // 数据
  pageSize: number;

  /** @deprecated v1.22 Phase 65 颜色自定义移除；后端契约保留，前端不再消费 */
  customPrimaryColor?: string;

  /** @deprecated v1.22 Phase 65 颜色自定义移除；后端契约保留，前端不再消费 */
  customSidebarColor?: string;

  // 语言
  language: string;
}

// ============= 默认配置 =============

/**
 * 默认主题配置
 */
export const defaultThemeConfiguration: ThemeConfiguration = {
  mode: "light",
};

/**
 * 默认布局配置
 */
export const defaultLayoutConfiguration: LayoutConfiguration = {
  type: "classic",
  sidebar: {
    collapsed: false,
    width: 280,
    collapsedWidth: 64,
  },
  density: "comfortable",
};

/**
 * 默认数据配置
 */
export const defaultDataConfiguration: DataConfiguration = {
  defaultPageSize: 10,
  pageSizeOptions: [10, 20, 50, 100],
};

/**
 * 完整的默认用户偏好设置
 */
export const defaultUserPreferences: UserPreferences = {
  version: 2,
  theme: defaultThemeConfiguration,
  layout: defaultLayoutConfiguration,
  data: defaultDataConfiguration,
  language: "zh-CN",
};

// ============= 配置验证辅助函数 =============

/**
 * 验证布局类型是否有效
 */
export function isValidLayoutType(type: string): type is LayoutType {
  return ["classic", "hybrid", "innovative"].includes(type);
}

/**
 * 验证密度模式是否有效
 */
export function isValidDensityMode(density: string): density is DensityMode {
  return ["compact", "comfortable", "spacious"].includes(density);
}

/**
 * 验证颜色模式是否有效
 */
export function isValidColorMode(mode: string): mode is ColorMode {
  return ["light", "dark"].includes(mode);
}
