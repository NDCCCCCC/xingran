/**
 * FloorPlanEditor 类型定义
 */

export interface WorkstationNode {
  id: string;
  code: string;
  name: string;
  x: number;  // positionX
  y: number;  // positionY
  width: number;   // 默认 80
  height: number;  // 默认 60
  status: number;
  type: number;  // workstationType
  rotation?: number;  // 旋转角度（度），默认0
}

export interface FloorPlanEditorProps {
  floorId: string;
  workstations: WorkstationNode[];
  onUpdatePosition: (items: {id: string; positionX: number; positionY: number; rotation?: number}[]) => Promise<void>;
  onEdit: (workstation: WorkstationNode) => void;
}

export interface ViewState {
  scale: number;
  offsetX: number;
  offsetY: number;
  isDragging: boolean;
  dragStartX: number;
  dragStartY: number;
}

export interface DragState {
  isDragging: boolean;
  workstationId: string | null;
  startX: number;
  startY: number;
  originalX: number;
  originalY: number;
}

export interface ContextMenuState {
  visible: boolean;
  x: number;
  y: number;
  workstation: WorkstationNode | null;
}
