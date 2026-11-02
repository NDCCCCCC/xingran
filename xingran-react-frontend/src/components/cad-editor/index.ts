/**
 * CAD 编辑器组件统一导出
 */

// ==================== 组件导出 ====================
export { CADFloorPlanEditor } from "./CADFloorPlanEditor";
export { CADToolbar } from "./CADToolbar";
export { CADPropertyPanel } from "./CADPropertyPanel";
export { CADLayersPanel } from "./CADLayersPanel";

// ==================== 组件 Props 类型导出 ====================
export type { CADFloorPlanEditorProps } from "./CADFloorPlanEditor";
export type { CADToolbarProps } from "./CADToolbar";
export type { CADPropertyPanelProps } from "./CADPropertyPanel";
export type { CADLayersPanelProps } from "./CADLayersPanel";

// ==================== 数据类型导出 ====================
export type {
  Point,
  Size,
  Wall,
  WallType,
  Door,
  DoorType,
  DoorDirection,
  TextElement,
  FloorPlanData,
} from "./types";

// ==================== 状态类型导出 ====================
export type {
  EditorMode,
  SelectedElementType,
  EditorState,
  ViewState,
  HistoryItem,
  HistoryState,
} from "./types";

// ==================== UI 配置类型导出 ====================
export type {
  ToolAction,
  Layer,
  LayerType,
  GridConfig,
  GuideLine,
  GuideLineType,
  ElementProperties,
} from "./types";

// ==================== 导出类型导出 ====================
export type { ExportFormat, ExportOptions } from "./types";

// ==================== 事件类型导出 ====================
export type { SelectEvent, UpdateEvent, CanvasClickEvent, DragEvent } from "./types";

// ==================== 主题导出 ====================
export { CAD_COLOR_THEME, getWallColor, getDoorColor, getWorkstationColor } from "./theme";
export type { CadColorTheme } from "./theme";
export { WORKSTATION_STATUS, DEFAULT_SNAP_DISTANCE, DEFAULT_GRID_SIZE } from "./theme";
