/**
 * 楼层管理页面常量
 */

// 视图模式
export type ViewMode = "table" | "card";
export type PageMode = "list" | "editor";

// 工位布局默认值（毫米）
export const WORKSTATION_LAYOUT = {
  DEFAULT_WIDTH: 160,
  DEFAULT_DEPTH: 70,
  GAP: 120,
  ITEMS_PER_ROW: 5,
  START_X: 100,
  START_Y: 100,
} as const;

// 平面图默认配置
export const DEFAULT_FLOOR_PLAN_CONFIG = {
  CANVAS_WIDTH: 2000,
  CANVAS_HEIGHT: 2000,
  GRID_SIZE: 20,
} as const;

// 状态选项
export const STATUS_OPTIONS = [
  { label: "正常", value: 0 },
  { label: "停用", value: 1 },
] as const;

// 表单状态默认值
export const DEFAULT_FORM_VALUES = {
  status: 0,
} as const;
