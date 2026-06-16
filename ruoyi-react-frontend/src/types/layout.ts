/**
 * 布局系统类型定义
 */

export type LayoutType = "classic" | "hybrid" | "innovative";

/**
 * 侧边栏配置
 */
export interface SidebarConfig {
  collapsible: boolean;
  width: number;
  collapsedWidth: number;
  position: "left" | "right";
  defaultCollapsed?: boolean;
}

/**
 * 顶部栏配置
 */
export interface HeaderConfig {
  fixed: boolean;
  height: number;
  showBreadcrumb: boolean;
  showUserInfo: boolean;
  transparent?: boolean;
}

/**
 * 标签页配置
 */
export interface TabsConfig {
  enabled: boolean;
  position: "top" | "bottom" | "left" | "right";
  closable: boolean;
  draggable: boolean;
  maxTabs?: number;
  persist: boolean;
}

/**
 * 内容区配置
 */
export interface ContentConfig {
  padding: string;
  maxWidth?: string;
  centered: boolean;
  scrollable: boolean;
}

/**
 * 布局配置
 */
export interface LayoutConfig {
  id: LayoutType;
  name: string;
  description: string;
  features: {
    sidebar: SidebarConfig;
    header: HeaderConfig;
    tabs: TabsConfig;
    content: ContentConfig;
  };
}

/**
 * 标签页项
 */
export interface TabItem {
  key: string;
  title: string;
  path: string;
  closable: boolean;
  icon?: React.ReactNode;
  pinned?: boolean;
}

/**
 * 标签页状态
 */
export interface TabsState {
  tabs: TabItem[];
  activeTab: string;
  history: string[];
}

/**
 * 布局状态
 */
export interface LayoutState {
  currentLayout: LayoutType;
  sidebarCollapsed: boolean;
  headerVisible: boolean;
}

/**
 * 密度模式
 */
export type DensityMode = "compact" | "comfortable" | "spacious";

/**
 * 密度配置
 */
export interface DensityConfig {
  mode: DensityConfig;
  spacing: {
    compact: string;
    comfortable: string;
    spacious: string;
  };
  fontSize: {
    compact: string;
    comfortable: string;
    spacious: string;
  };
}
