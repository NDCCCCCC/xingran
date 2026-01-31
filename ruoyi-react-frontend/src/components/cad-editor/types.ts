/**
 * CAD 编辑器类型定义
 */

import type { WorkstationNode } from '@/components/shared/FloorPlanEditor.types';
import type { Point, Size } from '@/utils/cad/geometry';

export type { Point, Size };

// ==================== 基础类型 ====================

export type WallType = 'straight' | 'curved' | 'l_shaped' | 'polyline';
export type DoorType = 'single' | 'double' | 'sliding' | 'revolving' | 'emergency';
export type DoorDirection = 'left' | 'right' | 'double' | 'sliding';
export type EditorMode = 'select' | 'draw_wall' | 'draw_door' | 'draw_workstation' | 'draw_text';
export type LayerType = 'wall' | 'door' | 'workstation' | 'text' | 'grid' | 'guide';
export type ExportFormat = 'png' | 'jpg' | 'svg' | 'json';
export type GuideLineType = 'horizontal' | 'vertical' | 'angular';
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

export type SelectedElementType = 'wall' | 'door' | 'workstation' | null;

// ==================== 数据模型 ====================

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

export interface EditorState {
  mode: EditorMode;
  selectedId: string | null;
  selectedType: SelectedElementType;
  isDrawing: boolean;
  scale: number;
  offset: Point;
}

export interface ViewState {
  scale: number;
  offset: Point;
  minScale?: number;
  maxScale?: number;
}

export interface HistoryItem {
  id: string;
  type: 'create' | 'update' | 'delete';
  elementType: 'wall' | 'door' | 'workstation';
  data: unknown;
  timestamp: number;
}

export interface HistoryState {
  past: HistoryItem[];
  present: FloorPlanData;
  future: HistoryItem[];
}

// ==================== UI 组件 ====================

export interface Layer {
  id: string;
  name: string;
  type: LayerType;
  visible: boolean;
  locked: boolean;
  opacity: number;
}

export interface GridConfig {
  size: number;
  color: string;
  showMajorLines: boolean;
  majorLineInterval: number;
  snapToGrid: boolean;
}

export interface GuideLine {
  id: string;
  type: GuideLineType;
  position: number;
  angle?: number;
  visible: boolean;
}

export interface ElementProperties {
  id: string;
  type: 'wall' | 'door' | 'workstation';
  properties: Record<string, unknown>;
}

// ==================== 导出配置 ====================

export interface ExportOptions {
  format: ExportFormat;
  quality?: number;
  scale?: number;
  includeGrid?: boolean;
  includeDimensions?: boolean;
}

// ==================== 事件类型 ====================

export interface SelectEvent {
  elementId: string;
  elementType: 'wall' | 'door' | 'workstation';
  ctrlKey?: boolean;
  shiftKey?: boolean;
}

export interface UpdateEvent {
  elementId: string;
  elementType: 'wall' | 'door' | 'workstation';
  changes: Record<string, unknown>;
}

export interface CanvasClickEvent {
  point: Point;
  button: 'left' | 'middle' | 'right';
  ctrlKey: boolean;
  shiftKey: boolean;
  altKey: boolean;
}

export interface DragEvent {
  elementId: string;
  elementType: 'wall' | 'door' | 'workstation';
  from: Point;
  to: Point;
}
