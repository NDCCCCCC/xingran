/**
 * CAD 编辑器组件导出
 */

export { CADFloorPlanEditor } from './CADFloorPlanEditor';
export { CADToolbar } from './CADToolbar';
export { CADPropertyPanel } from './CADPropertyPanel';
export { CADLayersPanel } from './CADLayersPanel';

// 导出类型
export type {
  Point,
  Size,
  Wall,
  WallType,
  Door,
  DoorType,
  DoorDirection,
  FloorPlanData,
  EditorMode,
  SelectedElementType,
  EditorState,
  ViewState,
  ToolAction,
  Layer,
  LayerType,
  HistoryItem,
  HistoryState,
  ElementProperties,
  GridConfig,
  ExportFormat,
  ExportOptions,
  GuideLine,
  GuideLineType,
  SelectEvent,
  UpdateEvent,
  CanvasClickEvent,
  DragEvent,
} from './types';

// 导出主题
export { CAD_COLOR_THEME, getWallColor, getDoorColor, getWorkstationColor } from './theme';

// 导出接口
export type { CADFloorPlanEditorProps } from './CADFloorPlanEditor';
export type { CADToolbarProps } from './CADToolbar';
export type { CADPropertyPanelProps } from './CADPropertyPanel';
export type { CADLayersPanelProps } from './CADLayersPanel';
