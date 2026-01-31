/**
 * CAD 编辑器类型定义
 */

import type { WorkstationNode } from '@/components/shared/FloorPlanEditor.types';
import type { Point, Size } from '@/utils/cad/geometry';

export type { Point, Size };

// ==================== 枚举类型 ====================

/** 墙体类型 */
export type WallType = 'straight' | 'curved' | 'l_shaped' | 'polyline';

/** 门类型 */
export type DoorType = 'single' | 'double' | 'sliding' | 'revolving' | 'emergency';

/** 门开启方向 */
export type DoorDirection = 'left' | 'right' | 'double' | 'sliding';

/** 编辑器模式 */
export type EditorMode = 'select' | 'draw_wall' | 'draw_door' | 'draw_workstation' | 'draw_text';

/** 图层类型 */
export type LayerType = 'wall' | 'door' | 'workstation' | 'text' | 'grid' | 'guide';

/** 导出格式 */
export type ExportFormat = 'png' | 'jpg' | 'svg' | 'json';

/** 辅助线类型 */
export type GuideLineType = 'horizontal' | 'vertical' | 'angular';

/** 工具操作 */
export type ToolAction =
  | 'select'
  | 'draw_wall'
  | 'draw_door'
  | 'draw_workstation'
  | 'zoom_in'
  | 'zoom_out'
  | 'reset_view'
  | 'save'
  | 'undo'
  | 'redo';

/** 选中元素类型 */
export type SelectedElementType = 'wall' | 'door' | 'workstation' | null;

// ==================== 数据模型 ====================

/** 墙体数据 */
export interface Wall {
  id: string;
  floorId: string;
  type: WallType;
  points: Point[];
  thickness: number;
  height: number;
  color: string;
  name?: string;
  remark?: string;
}

/** 门数据 */
export interface Door {
  id: string;
  floorId: string;
  wallId?: string;
  position: Point;
  angle: number;
  type: DoorType;
  direction: DoorDirection;
  width: number;
  length: number;
  color: string;
  name?: string;
  remark?: string;
}

/** 文字元素 */
export interface TextElement {
  id: string;
  floorId: string;
  position: Point;
  content: string;
  fontSize: number;
  color: string;
  fontFamily?: string;
  fontWeight?: 'normal' | 'bold';
  fontStyle?: 'normal' | 'italic';
  angle?: number;
}

/** 平面图数据 */
export interface FloorPlanData {
  floorId: string;
  floorName: string;
  width: number;
  height: number;
  walls: Wall[];
  doors: Door[];
  workstations: WorkstationNode[];
  texts?: TextElement[];
  gridSize?: number;
  showGrid?: boolean;
  snapToGrid?: boolean;
}

// ==================== 状态管理 ====================

/** 编辑器状态 */
export interface EditorState {
  mode: EditorMode;
  selectedId: string | null;
  selectedType: SelectedElementType;
  isDrawing: boolean;
  scale: number;
  offset: Point;
}

/** 视图状态 */
export interface ViewState {
  scale: number;
  offset: Point;
  minScale?: number;
  maxScale?: number;
}

/** 历史记录项 */
export interface HistoryItem {
  id: string;
  type: 'create' | 'update' | 'delete';
  elementType: 'wall' | 'door' | 'workstation';
  data: unknown;
  timestamp: number;
}

/** 历史记录状态 */
export interface HistoryState {
  past: HistoryItem[];
  present: FloorPlanData;
  future: HistoryItem[];
}

// ==================== UI 组件配置 ====================

/** 图层配置 */
export interface Layer {
  id: string;
  name: string;
  type: LayerType;
  visible: boolean;
  locked: boolean;
  opacity: number;
}

/** 网格配置 */
export interface GridConfig {
  size: number;
  color: string;
  showMajorLines: boolean;
  majorLineInterval: number;
  snapToGrid: boolean;
}

/** 辅助线 */
export interface GuideLine {
  id: string;
  type: GuideLineType;
  position: number;
  angle?: number;
  visible: boolean;
}

/** 元素属性 */
export interface ElementProperties {
  id: string;
  type: 'wall' | 'door' | 'workstation';
  properties: Record<string, unknown>;
}

// ==================== 导出配置 ====================

/** 导出选项 */
export interface ExportOptions {
  format: ExportFormat;
  quality?: number;
  scale?: number;
  includeGrid?: boolean;
  includeDimensions?: boolean;
}

// ==================== 事件类型 ====================

/** 选择事件 */
export interface SelectEvent {
  elementId: string;
  elementType: 'wall' | 'door' | 'workstation';
  ctrlKey?: boolean;
  shiftKey?: boolean;
}

/** 更新事件 */
export interface UpdateEvent {
  elementId: string;
  elementType: 'wall' | 'door' | 'workstation';
  changes: Record<string, unknown>;
}

/** 画布点击事件 */
export interface CanvasClickEvent {
  point: Point;
  button: 'left' | 'middle' | 'right';
  ctrlKey: boolean;
  shiftKey: boolean;
  altKey: boolean;
}

/** 拖拽事件 */
export interface DragEvent {
  elementId: string;
  elementType: 'wall' | 'door' | 'workstation';
  from: Point;
  to: Point;
}
