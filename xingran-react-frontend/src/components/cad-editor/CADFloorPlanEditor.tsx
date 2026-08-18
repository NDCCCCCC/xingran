/**
 * CAD 楼层平面图编辑器主组件
 */

import { useState, useCallback, useRef, useEffect, useMemo } from "react";
import { App, Modal, Input } from "antd";
import { CADToolbar } from "./CADToolbar";
import { CADPropertyPanel } from "./CADPropertyPanel";
import { CADLayersPanel } from "./CADLayersPanel";
import { WallElement, DoorElement, WorkstationElement, CADTextElement } from "../cad-elements";
import { useWallDrawing, type WallDrawingResult } from "@/hooks/useWallDrawing";
import { CAD_COLOR_THEME } from "./theme";
import {
  EDITOR_CONSTANTS,
  snapToGrid,
  checkWorkstationCollision,
  pointToLineDistance,
} from "./editorUtils";
import type {
  FloorPlanData,
  Wall,
  Door,
  EditorMode,
  Layer,
  LayerConfig,
  Point,
  WallType,
  TextElement,
} from "./types";
import type { WorkstationNode } from "@/components/shared/FloorPlanEditor.types";

const DRAG_THRESHOLD = 5;

export interface CADFloorPlanEditorProps {
  floorId: string;
  floorName: string;
  walls?: Wall[];
  doors?: Door[];
  workstations?: WorkstationNode[];
  texts?: TextElement[];
  planImageId?: string; // 平面图图片ID
  planImageUrl?: string; // 平面图图片URL
  layerConfig?: LayerConfig; // 图层配置
  onSave?: (data: FloorPlanData) => void;
  readOnly?: boolean;
  style?: React.CSSProperties;
}

export function CADFloorPlanEditor({
  floorId,
  floorName,
  walls = [],
  doors = [],
  workstations = [],
  texts = [],
  planImageId,
  planImageUrl,
  layerConfig,
  onSave,
  readOnly = false,
  style,
}: CADFloorPlanEditorProps) {
  // ==================== 状态 ====================
  const { message } = App.useApp();
  const [mode, setMode] = useState<EditorMode>("select");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [selectedType, setSelectedType] = useState<"wall" | "door" | "workstation" | "text" | null>(
    null
  );
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [hoveredId, setHoveredId] = useState<string | null>(null);
  const [scale, setScale] = useState(1);
  const [offset, setOffset] = useState<Point>({ x: 0, y: 0 });
  const [history, _setHistory] = useState<FloorPlanData[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const [isTextInputVisible, setIsTextInputVisible] = useState(false);
  const [textInputPosition, setTextInputPosition] = useState<Point | null>(null);
  const [tempTextContent, setTempTextContent] = useState("");

  // 图层状态
  const [layers, setLayers] = useState<Layer[]>([
    { id: "grid", name: "网格", type: "grid", visible: true, locked: true, opacity: 1 },
    {
      id: "plan_image",
      name: "平面图",
      type: "plan_image",
      visible: true,
      locked: true,
      opacity: 0.5,
    },
    { id: "wall", name: "墙体", type: "wall", visible: true, locked: false, opacity: 1 },
    { id: "door", name: "门", type: "door", visible: true, locked: false, opacity: 1 },
    {
      id: "workstation",
      name: "工位",
      type: "workstation",
      visible: true,
      locked: false,
      opacity: 1,
    },
    { id: "text", name: "文本", type: "text", visible: true, locked: false, opacity: 1 },
  ]);

  // 当前平面图数据
  const [floorPlanData, setFloorPlanData] = useState<FloorPlanData>({
    floorId,
    floorName,
    width: EDITOR_CONSTANTS.DEFAULT_CANVAS_WIDTH,
    height: EDITOR_CONSTANTS.DEFAULT_CANVAS_HEIGHT,
    walls,
    doors,
    workstations,
    texts: texts || [],
    planImageId,
    planImageUrl,
    planImageTransform: { x: 0, y: 0, scale: 1 },
    layerConfig,
    gridSize: EDITOR_CONSTANTS.DEFAULT_GRID_SIZE,
    showGrid: true,
    snapToGrid: true,
  });

  // 拖动和交互状态
  const svgRef = useRef<SVGSVGElement>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [_dragStart, setDragStart] = useState<Point | null>(null);
  const [lastMousePos, setLastMousePos] = useState<Point | null>(null);
  const [isAltPressed, setIsAltPressed] = useState(false);
  const lastElementMouseCanvasPos = useRef<Point | null>(null);
  const dragStartCanvasPos = useRef<Point | null>(null);
  const hasMovedBeyondThreshold = useRef<boolean>(false);
  // 存储拖动开始时工位的原始位置
  const originalWorkstationPositions = useRef<Map<string, { x: number; y: number }>>(new Map());
  const [draggedElement, setDraggedElement] = useState<{
    id: string;
    type: "wall" | "door" | "workstation" | "text";
  } | null>(null);
  const [isBoxSelecting, setIsBoxSelecting] = useState(false);
  const [boxSelectStart, setBoxSelectStart] = useState<Point | null>(null);
  const [boxSelectEnd, setBoxSelectEnd] = useState<Point | null>(null);
  const [editingWallId, setEditingWallId] = useState<string | null>(null);

  // 平面图图片拖拽状态
  const [isDraggingPlanImage, setIsDraggingPlanImage] = useState(false);
  const planImageDragStart = useRef<Point | null>(null);

  // 使用 ref 存储 offset 和 scale 的最新值，避免闭包问题
  const offsetRef = useRef(offset);
  const scaleRef = useRef(scale);

  // 同步 ref 值
  useEffect(() => {
    offsetRef.current = offset;
  }, [offset]);

  useEffect(() => {
    scaleRef.current = scale;
  }, [scale]);

  // ==================== 墙体绘制 ====================
  function getWallConnectionPoints(
    nearbyNode: { point: Point; wallId: string; pointIndex: number } | null
  ): Point[] | null {
    if (!nearbyNode) return null;

    const wall = floorPlanData.walls.find((w) => w.id === nearbyNode.wallId);
    if (!wall) return null;

    // 根据点击的节点位置，决定绘制方向
    if (nearbyNode.pointIndex === 0) {
      // 点击的是起点，从头开始，反转点数组以便从起点向外绘制
      return [...wall.points].reverse();
    } else if (nearbyNode.pointIndex === wall.points.length - 1) {
      // 点击的是终点，从尾部继续
      return [...wall.points];
    } else {
      // 点击的是中间节点，使用从该节点到终点的点
      return wall.points.slice(nearbyNode.pointIndex);
    }
  }

  // ==================== 墙体绘制 ====================
  const wallDrawing = useWallDrawing({
    gridSize: floorPlanData.gridSize,
    snapEnabled: floorPlanData.snapToGrid,
    onComplete: useCallback(
      (wall: WallDrawingResult) => {
        if (editingWallId) {
          // 更新已有的墙体
          setFloorPlanData((prev) => ({
            ...prev,
            walls: prev.walls.map((w) =>
              w.id === editingWallId
                ? { ...w, points: wall.points, type: wall.type as WallType }
                : w
            ),
          }));
          message.success("墙体已更新");
          setEditingWallId(null);
        } else {
          // 创建新墙体
          const newWall: Wall = {
            id: `wall_${Date.now()}`,
            floorId,
            type: wall.type as WallType,
            points: wall.points,
            thickness: 10,
            height: 3.0,
            color: CAD_COLOR_THEME.wall.default,
          };
          setFloorPlanData((prev) => ({
            ...prev,
            walls: [...prev.walls, newWall],
          }));
          message.success("墙体绘制完成");
        }
      },
      // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
      [floorId, editingWallId]
    ),
  });

  // ==================== 门绘制 ====================
  const [_isDrawingDoor, _setIsDrawingDoor] = useState(false);

  // ==================== 图层操作 ====================
  const handleLayerVisibilityChange = useCallback((layerId: string, visible: boolean) => {
    setLayers((prev) =>
      prev.map((layer) => (layer.id === layerId ? { ...layer, visible } : layer))
    );
  }, []);

  const handleLayerLockChange = useCallback((layerId: string, locked: boolean) => {
    setLayers((prev) => prev.map((layer) => (layer.id === layerId ? { ...layer, locked } : layer)));
  }, []);

  const handleLayerOpacityChange = useCallback((layerId: string, opacity: number) => {
    setLayers((prev) =>
      prev.map((layer) => (layer.id === layerId ? { ...layer, opacity } : layer))
    );
  }, []);

  const selectedElement = useMemo(() => {
    if (!selectedId || !selectedType) return null;
    if (selectedType === "wall") {
      return floorPlanData.walls.find((w) => w.id === selectedId) ?? null;
    }
    if (selectedType === "door") {
      return floorPlanData.doors.find((d) => d.id === selectedId) ?? null;
    }
    if (selectedType === "workstation") {
      return floorPlanData.workstations.find((ws) => ws.id === selectedId) ?? null;
    }
    if (selectedType === "text") {
      return floorPlanData.texts?.find((t) => t.id === selectedId) ?? null;
    }
    return null;
  }, [selectedId, selectedType, floorPlanData]);

  // ==================== 视图操作 ====================
  const handleZoomIn = useCallback(() => {
    setScale((prev) => Math.min(prev + EDITOR_CONSTANTS.ZOOM_STEP, EDITOR_CONSTANTS.MAX_SCALE));
  }, []);

  const handleZoomOut = useCallback(() => {
    setScale((prev) => Math.max(prev - EDITOR_CONSTANTS.ZOOM_STEP, EDITOR_CONSTANTS.MIN_SCALE));
  }, []);

  const handleResetView = useCallback(() => {
    setScale(1);
    setOffset({ x: 0, y: 0 });
  }, []);

  // 平面图自动隐藏逻辑：当有墙体或门时，默认隐藏平面图
  useEffect(() => {
    const hasWallsOrDoors = floorPlanData.walls.length > 0 || floorPlanData.doors.length > 0;
    const planImageLayer = layers.find((l) => l.id === "plan_image");

    // 如果有图层配置，使用配置值；否则使用默认行为（有内容时隐藏）
    const autoHide = layerConfig?.planImage?.autoHide !== false;

    if (autoHide && planImageLayer) {
      setLayers((prev) =>
        prev.map((layer) =>
          layer.id === "plan_image" ? { ...layer, visible: !hasWallsOrDoors } : layer
        )
      );
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- layers is state array; disable to avoid re-run loop
  }, [floorPlanData.walls.length, floorPlanData.doors.length, layerConfig?.planImage?.autoHide]);

  // ==================== 图层辅助函数 ====================
  function isLayerLocked(layerId: string): boolean {
    return layers.find((l) => l.id === layerId)?.locked ?? false;
  }

  // 鼠标滚轮缩放（Shift+滚轮缩放平面图，普通滚轮缩放画布）
  useEffect(() => {
    const svgElement = svgRef.current;
    if (!svgElement) return;

    const handleWheel = (e: WheelEvent) => {
      e.preventDefault();

      // 检测是否在平面图图片上且按住 Shift 键
      const target = e.target as SVGElement;
      const isOnPlanImage = target.closest('g[data-plan-image="true"]');
      const isShiftKey = (e as WheelEvent).shiftKey;

      if (
        isOnPlanImage &&
        isShiftKey &&
        !readOnly &&
        !isLayerLocked("plan_image") &&
        floorPlanData.planImageUrl
      ) {
        // 缩放平面图图片
        const delta = e.deltaY > 0 ? -0.1 : 0.1;
        setFloorPlanData((prev) => ({
          ...prev,
          planImageTransform: {
            ...(prev.planImageTransform || { x: 0, y: 0, scale: 1 }),
            scale: Math.max(0.1, Math.min(5, (prev.planImageTransform?.scale || 1) + delta)),
          },
        }));
        return;
      }

      // 否则缩放画布
      const zoomDelta = e.deltaY > 0 ? -EDITOR_CONSTANTS.ZOOM_STEP : EDITOR_CONSTANTS.ZOOM_STEP;
      const prevScale = scaleRef.current;
      const newScale = Math.max(
        EDITOR_CONSTANTS.MIN_SCALE,
        Math.min(EDITOR_CONSTANTS.MAX_SCALE, prevScale + zoomDelta)
      );

      const rect = svgRef.current?.getBoundingClientRect();
      if (rect) {
        const mouseX = e.clientX - rect.left;
        const mouseY = e.clientY - rect.top;
        const currentOffset = offsetRef.current;
        const canvasX = (mouseX - currentOffset.x) / prevScale;
        const canvasY = (mouseY - currentOffset.y) / prevScale;
        const newOffsetX = mouseX - canvasX * newScale;
        const newOffsetY = mouseY - canvasY * newScale;
        setOffset({ x: newOffsetX, y: newOffsetY });
      }

      setScale(newScale);
    };

    svgElement.addEventListener("wheel", handleWheel, { passive: false });

    return () => {
      svgElement.removeEventListener("wheel", handleWheel);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- isLayerLocked is a render-defined function
  }, [readOnly, floorPlanData.planImageUrl]);

  // ==================== 辅助函数 ====================
  function getElementTypeById(id: string): "wall" | "door" | "workstation" | "text" | null {
    if (floorPlanData.walls.find((w) => w.id === id)) return "wall";
    if (floorPlanData.doors.find((d) => d.id === id)) return "door";
    if (floorPlanData.workstations.find((w) => w.id === id)) return "workstation";
    if (floorPlanData.texts?.find((t) => t.id === id)) return "text";
    return null;
  }

  // ==================== 选择操作 ====================
  const handleSelectElement = useCallback(
    (id: string, type: "wall" | "door" | "workstation" | "text", addToSelection = false) => {
      if (addToSelection) {
        // 多选模式
        setSelectedIds((prev) => {
          const newSet = new Set(prev);
          if (newSet.has(id)) {
            newSet.delete(id);
          } else {
            newSet.add(id);
          }
          return newSet;
        });
        // 保持最后选中的元素作为当前选中
        setSelectedId(id);
        setSelectedType(type);
      } else {
        // 单选模式
        setSelectedIds(new Set([id]));
        setSelectedId(id);
        setSelectedType(type);
      }
    },
    []
  );

  const handleDeselect = useCallback(() => {
    setSelectedId(null);
    setSelectedType(null);
    setSelectedIds(new Set());
  }, []);

  // 获取当前选中的所有元素
  const _selectedElements = useMemo(() => {
    const elements: { id: string; type: "wall" | "door" | "workstation" | "text" }[] = [];
    if (selectedIds.size === 0) return elements;

    for (const id of selectedIds) {
      const wall = floorPlanData.walls.find((w) => w.id === id);
      if (wall) {
        elements.push({ id: wall.id, type: "wall" });
        continue;
      }
      const door = floorPlanData.doors.find((d) => d.id === id);
      if (door) {
        elements.push({ id: door.id, type: "door" });
        continue;
      }
      const ws = floorPlanData.workstations.find((w) => w.id === id);
      if (ws) {
        elements.push({ id: ws.id, type: "workstation" });
        continue;
      }
      const text = floorPlanData.texts?.find((t) => t.id === id);
      if (text) {
        elements.push({ id: text.id, type: "text" });
      }
    }
    return elements;
  }, [selectedIds, floorPlanData]);

  // ==================== 保存 ====================
  const handleSave = useCallback(() => {
    onSave?.(floorPlanData);
    message.success("保存成功");
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [floorPlanData, onSave]);

  // ==================== 更新元素属性 ====================
  const handleUpdateElement = useCallback(
    (changes: Partial<Wall | Door | WorkstationNode>) => {
      if (!selectedId || !selectedType) return;

      setFloorPlanData((prev) => {
        const newData = { ...prev };
        if (selectedType === "wall") {
          newData.walls = prev.walls.map((w) =>
            w.id === selectedId ? ({ ...w, ...changes } as Wall) : w
          );
        } else if (selectedType === "door") {
          newData.doors = prev.doors.map((d) =>
            d.id === selectedId ? ({ ...d, ...changes } as Door) : d
          );
        } else if (selectedType === "workstation") {
          newData.workstations = prev.workstations.map((ws) =>
            ws.id === selectedId ? ({ ...ws, ...changes } as WorkstationNode) : ws
          );
        }
        return newData;
      });
    },
    [selectedId, selectedType]
  );

  // ==================== 撤销/重做 ====================
  const canUndo = historyIndex > 0;
  const canRedo = historyIndex < history.length - 1;

  const handleUndo = useCallback(() => {
    if (canUndo) {
      const newIndex = historyIndex - 1;
      setHistoryIndex(newIndex);
      setFloorPlanData(history[newIndex]);
    }
  }, [canUndo, history, historyIndex]);

  const handleRedo = useCallback(() => {
    if (canRedo) {
      const newIndex = historyIndex + 1;
      setHistoryIndex(newIndex);
      setFloorPlanData(history[newIndex]);
    }
  }, [canRedo, history, historyIndex]);

  // ==================== 文本输入 ====================
  function closeTextInput() {
    setIsTextInputVisible(false);
    setTextInputPosition(null);
    setTempTextContent("");
  }

  const handleTextInputOk = useCallback(() => {
    if (!textInputPosition || !tempTextContent.trim()) {
      closeTextInput();
      return;
    }

    const newText: TextElement = {
      id: `text_${Date.now()}`,
      floorId,
      position: textInputPosition,
      content: tempTextContent.trim(),
      fontSize: 14,
      color: "var(--theme-text-primary, #333333)",
    };

    setFloorPlanData((prev) => ({
      ...prev,
      texts: [...(prev.texts || []), newText],
    }));

    closeTextInput();
    message.success("文本已添加");
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [textInputPosition, tempTextContent, floorId]);

  const handleTextInputCancel = useCallback(() => {
    closeTextInput();
  }, []);

  // ==================== 坐标转换 ====================
  function isLayerVisible(layerId: string): boolean {
    return layers.find((l) => l.id === layerId)?.visible ?? false;
  }

  function getLayerOpacity(layerId: string): number {
    return layers.find((l) => l.id === layerId)?.opacity ?? 1;
  }

  // ==================== 坐标转换 ====================
  const getCanvasPoint = useCallback(
    (clientX: number, clientY: number): Point => {
      const rect = svgRef.current?.getBoundingClientRect();
      if (!rect) return { x: 0, y: 0 };
      return {
        x: (clientX - rect.left - offset.x) / scale,
        y: (clientY - rect.top - offset.y) / scale,
      };
    },
    [scale, offset]
  );

  // 检测是否点击在元素上
  const getHitElement = useCallback(
    (point: Point): { id: string; type: "wall" | "door" | "workstation" | "text" } | null => {
      // 检查文本（最上层）
      for (const text of floorPlanData.texts || []) {
        const textWidth = text.content.length * text.fontSize * 0.6;
        const textHeight = text.fontSize;
        if (
          point.x >= text.position.x &&
          point.x <= text.position.x + textWidth &&
          point.y >= text.position.y &&
          point.y <= text.position.y + textHeight
        ) {
          return { id: text.id, type: "text" };
        }
      }
      // 检查工位（优先检查，因为工位通常在最上层）
      for (const ws of floorPlanData.workstations) {
        const halfW = (ws.width || 120) / 2;
        const halfH = (ws.height || 80) / 2;
        // 简单的矩形碰撞检测（不考虑旋转）
        if (
          point.x >= ws.x - halfW &&
          point.x <= ws.x + halfW &&
          point.y >= ws.y - halfH &&
          point.y <= ws.y + halfH
        ) {
          return { id: ws.id, type: "workstation" };
        }
      }
      // 检查门
      for (const door of floorPlanData.doors) {
        const doorPos = door.position;
        if (typeof doorPos === "object" && "x" in doorPos) {
          const dx = point.x - doorPos.x;
          const dy = point.y - doorPos.y;
          if (Math.sqrt(dx * dx + dy * dy) < 30) {
            return { id: door.id, type: "door" };
          }
        }
      }
      // 检查墙体（放在最后，因为墙体通常在底层）
      for (const wall of floorPlanData.walls) {
        if (wall.points.length < 2) continue;
        for (let i = 0; i < wall.points.length - 1; i++) {
          const p1 = wall.points[i];
          const p2 = wall.points[i + 1];
          const dist = pointToLineDistance(point, p1, p2);
          if (dist < wall.thickness / 2 + 5) {
            return { id: wall.id, type: "wall" };
          }
        }
      }
      return null;
    },
    [floorPlanData]
  );

  // 检测点击位置是否在已有的墙体节点附近
  const findNearbyWallNode = useCallback(
    (point: Point, threshold = 15): { point: Point; wallId: string; pointIndex: number } | null => {
      for (const wall of floorPlanData.walls) {
        for (let i = 0; i < wall.points.length; i++) {
          const node = wall.points[i];
          const dist = Math.sqrt(Math.pow(point.x - node.x, 2) + Math.pow(point.y - node.y, 2));
          if (dist < threshold) {
            return { point: node, wallId: wall.id, pointIndex: i };
          }
        }
      }
      return null;
    },
    [floorPlanData]
  );

  // ==================== 画布事件 ====================
  const handleCanvasMouseDown = useCallback(
    (e: React.MouseEvent) => {
      const point = getCanvasPoint(e.clientX, e.clientY);

      // 检查是否点击在元素上
      const hitElement = getHitElement(point);
      const layerIsLocked = hitElement ? isLayerLocked(hitElement.type) : false;

      // 中键或Alt+左键：总是平移（所有模式通用）
      if (e.button === 1 || (e.button === 0 && e.altKey)) {
        setIsDragging(true);
        setDragStart({ x: e.clientX, y: e.clientY });
        setLastMousePos({ x: e.clientX, y: e.clientY });
        e.preventDefault();
        return;
      }

      // 左键
      if (e.button === 0) {
        // 绘制墙体模式
        if (mode === "draw_wall") {
          // 检测是否接近已有的墙体节点
          const nearbyNode = findNearbyWallNode(point);
          const snapPoint = nearbyNode ? nearbyNode.point : point;

          if (wallDrawing.drawPoints.length === 0) {
            const startPoints = getWallConnectionPoints(nearbyNode);
            if (nearbyNode && startPoints) {
              // 连接到已有墙体节点
              setEditingWallId(nearbyNode.wallId);
              wallDrawing.startDrawing(snapPoint, startPoints);
            } else {
              // 创建新墙体
              setEditingWallId(null);
              wallDrawing.startDrawing(snapPoint);
            }
          } else {
            wallDrawing.addPoint(snapPoint);
          }
          return;
        }

        // 绘制门模式
        if (mode === "draw_door") {
          // 检查是否点击了已存在的门
          if (hitElement && hitElement.type === "door") {
            // 选中已存在的门
            if (!layerIsLocked && !readOnly) {
              handleSelectElement(hitElement.id, "door");
              // 获取完整的门对象用于拖动
              const fullDoor = floorPlanData.doors.find((d) => d.id === hitElement.id);
              if (fullDoor) {
                setDraggedElement({ id: fullDoor.id, type: "door" });
                lastElementMouseCanvasPos.current = point;
                dragStartCanvasPos.current = point;
                hasMovedBeyondThreshold.current = false;
              }
            }
          } else {
            // 创建新门
            const newDoor: Door = {
              id: `door_${Date.now()}`,
              floorId,
              position: point,
              angle: 0,
              type: "single",
              direction: "left",
              width: 80,
              length: 50,
              color: CAD_COLOR_THEME.door.default,
            };
            setFloorPlanData((prev) => ({
              ...prev,
              doors: [...prev.doors, newDoor],
            }));
            message.success("门已添加");
          }
          return;
        }

        // 绘制工位模式
        if (mode === "draw_workstation") {
          // 检查是否点击了已存在的工位
          if (hitElement && hitElement.type === "workstation") {
            // 选中已存在的工位
            if (!isLayerLocked("workstation") && !readOnly) {
              handleSelectElement(hitElement.id, "workstation");
              setDraggedElement(hitElement);
              lastElementMouseCanvasPos.current = point;
              dragStartCanvasPos.current = point;
              hasMovedBeyondThreshold.current = false;
              // 保存工位的原始位置
              const ws = floorPlanData.workstations.find((w) => w.id === hitElement.id);
              if (ws) {
                originalWorkstationPositions.current.set(ws.id, { x: ws.x, y: ws.y });
              }
            }
          } else {
            // 创建新工位 - 吸附到网格并检查碰撞
            const snappedX = floorPlanData.snapToGrid
              ? snapToGrid(point.x, floorPlanData.gridSize ?? EDITOR_CONSTANTS.DEFAULT_GRID_SIZE)
              : point.x;
            const snappedY = floorPlanData.snapToGrid
              ? snapToGrid(point.y, floorPlanData.gridSize ?? EDITOR_CONSTANTS.DEFAULT_GRID_SIZE)
              : point.y;

            const newWorkstation: WorkstationNode = {
              id: `ws_${Date.now()}`,
              code: `WS-${String(floorPlanData.workstations.length + 1).padStart(3, "0")}`,
              name: `工位-${floorPlanData.workstations.length + 1}`,
              x: snappedX,
              y: snappedY,
              width: 160, // 更宽的桌子
              height: 70, // 桌子深度
              rotation: 0,
              status: 0, // 0 = 空闲
              type: 0, // 0 = 一字型, 1 = L型
            };

            // 碰撞检测
            if (checkWorkstationCollision(newWorkstation, floorPlanData.workstations)) {
              message.warning("工位位置与其他工位冲突，请选择其他位置");
              return;
            }

            setFloorPlanData((prev) => ({
              ...prev,
              workstations: [...prev.workstations, newWorkstation],
            }));
            message.success("工位已添加");
          }
          return;
        }

        // 绘制文本模式
        if (mode === "draw_text") {
          // 检查是否点击了已存在的文本
          if (hitElement && hitElement.type === "text") {
            // 选中已存在的文本
            if (!isLayerLocked("text") && !readOnly) {
              handleSelectElement(hitElement.id, "text");
            }
          } else {
            // 显示文本输入对话框
            const snappedX = floorPlanData.snapToGrid
              ? snapToGrid(point.x, floorPlanData.gridSize ?? EDITOR_CONSTANTS.DEFAULT_GRID_SIZE)
              : point.x;
            const snappedY = floorPlanData.snapToGrid
              ? snapToGrid(point.y, floorPlanData.gridSize ?? EDITOR_CONSTANTS.DEFAULT_GRID_SIZE)
              : point.y;
            setTextInputPosition({ x: snappedX, y: snappedY });
            setTempTextContent("");
            setIsTextInputVisible(true);
          }
          return;
        }

        // 选择模式
        if (hitElement) {
          // 点击元素 - 选择并准备拖动
          const addToSelection = e.ctrlKey || e.shiftKey; // Ctrl/Shift 多选
          const wasSelected = selectedIds.has(hitElement.id); // 是否已经选中

          if (!isLayerLocked && !readOnly) {
            // 更新选中状态
            if (addToSelection) {
              // 添加到选择（多选模式）
              if (wasSelected) {
                // 从选择中移除
                const newSelectedIds = new Set(selectedIds);
                newSelectedIds.delete(hitElement.id);
                setSelectedIds(newSelectedIds);
                if (newSelectedIds.size === 0) {
                  handleDeselect();
                } else {
                  // 更新当前选中元素为选择集中的另一个
                  const lastId = Array.from(newSelectedIds).pop()!;
                  setSelectedId(lastId);
                }
              } else {
                // 添加到选择
                setSelectedIds((prev) => new Set(prev).add(hitElement.id));
                setSelectedId(hitElement.id);
                setSelectedType(hitElement.type);
              }
            } else if (!wasSelected) {
              // 单选且点击未选中的元素：只选中当前元素（清空之前的选择）
              setSelectedIds(new Set([hitElement.id]));
              setSelectedId(hitElement.id);
              setSelectedType(hitElement.type);
            }
            // 如果点击已选中的元素（单选模式），保持选择不变

            // 准备拖动状态，但不立即开始拖动（等待鼠标移动超过阈值）
            setDraggedElement({ id: hitElement.id, type: hitElement.type });
            lastElementMouseCanvasPos.current = point;
            dragStartCanvasPos.current = point;
            hasMovedBeyondThreshold.current = false;
            // 如果是工位，保存所有选中工位的原始位置
            if (hitElement.type === "workstation") {
              // 保存当前点击的工位
              const ws = floorPlanData.workstations.find((w) => w.id === hitElement.id);
              if (ws) {
                originalWorkstationPositions.current.set(ws.id, { x: ws.x, y: ws.y });
              }
              // 同时保存其他已选中的工位的原始位置
              selectedIds.forEach((id) => {
                if (id !== hitElement.id) {
                  const selectedWs = floorPlanData.workstations.find((w) => w.id === id);
                  if (selectedWs && !originalWorkstationPositions.current.has(id)) {
                    originalWorkstationPositions.current.set(id, {
                      x: selectedWs.x,
                      y: selectedWs.y,
                    });
                  }
                }
              });
            }
          } else {
            handleSelectElement(hitElement.id, hitElement.type, addToSelection);
          }
        } else {
          // 点击空白处 - 开始框选
          if (!e.ctrlKey && !e.shiftKey) {
            // 如果没有按 Ctrl/Shift，先清除选择
            handleDeselect();
          }
          // 开始框选
          setIsBoxSelecting(true);
          setBoxSelectStart(point);
          setBoxSelectEnd(point);
        }
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- mixed render-defined and useCallback deps; disable to avoid loop
    [
      mode,
      scale,
      offset,
      wallDrawing,
      getCanvasPoint,
      getHitElement,
      readOnly,
      handleSelectElement,
      handleDeselect,
      floorPlanData,
    ]
  );

  const handleCanvasMouseMove = useCallback(
    (e: React.MouseEvent) => {
      const point = getCanvasPoint(e.clientX, e.clientY);

      // 拖拽平面图图片
      if (isDraggingPlanImage && planImageDragStart.current && !readOnly) {
        const dx = (e.clientX - planImageDragStart.current.x) / scale;
        const dy = (e.clientY - planImageDragStart.current.y) / scale;
        setFloorPlanData((prev) => ({
          ...prev,
          planImageTransform: {
            ...(prev.planImageTransform || { x: 0, y: 0, scale: 1 }),
            x: (prev.planImageTransform?.x || 0) + dx,
            y: (prev.planImageTransform?.y || 0) + dy,
          },
        }));
        planImageDragStart.current = { x: e.clientX, y: e.clientY };
        return;
      }

      // 平移画布（Alt+左键或中键）
      if (isDragging && lastMousePos) {
        const dx = e.clientX - lastMousePos.x;
        const dy = e.clientY - lastMousePos.y;
        setOffset((prev) => ({ x: prev.x + dx, y: prev.y + dy }));
        setLastMousePos({ x: e.clientX, y: e.clientY });
        return;
      }

      // 框选模式
      if (isBoxSelecting && boxSelectStart) {
        setBoxSelectEnd(point);
        return;
      }

      // 批量拖动元素（检查是否有选中的元素）
      if ((draggedElement || selectedIds.size > 0) && dragStartCanvasPos.current && !readOnly) {
        // 检查是否超过拖动阈值
        if (!hasMovedBeyondThreshold.current) {
          const dx = point.x - dragStartCanvasPos.current.x;
          const dy = point.y - dragStartCanvasPos.current.y;
          const distance = Math.sqrt(dx * dx + dy * dy);
          if (distance < DRAG_THRESHOLD / scale) {
            // 未超过阈值，不执行拖动
            return;
          }
          // 超过阈值，标记为已开始拖动
          hasMovedBeyondThreshold.current = true;
        }

        // 使用上一次鼠标位置计算增量（避免累积误差）
        const lastPos = lastElementMouseCanvasPos.current;
        if (!lastPos) {
          // 首次移动，更新 ref 并返回
          lastElementMouseCanvasPos.current = point;
          return;
        }
        const dx = point.x - lastPos.x;
        const dy = point.y - lastPos.y;

        // 拖动所有选中的元素
        setFloorPlanData((prev) => {
          const newData = { ...prev };

          // 拖动墙体
          newData.walls = prev.walls.map((w) => {
            if (selectedIds.has(w.id)) {
              return {
                ...w,
                points: w.points.map((p) => ({ x: p.x + dx, y: p.y + dy })),
              };
            }
            return w;
          });

          // 拖动门
          newData.doors = prev.doors.map((d) => {
            if (selectedIds.has(d.id)) {
              return {
                ...d,
                position: { x: d.position.x + dx, y: d.position.y + dy },
              };
            }
            return d;
          });

          // 拖动工位（带网格吸附，无碰撞检测）
          newData.workstations = prev.workstations.map((ws) => {
            if (selectedIds.has(ws.id)) {
              // 先应用增量
              let newX = ws.x + dx;
              let newY = ws.y + dy;

              // 如果启用了网格吸附，实时吸附到网格
              if (floorPlanData.snapToGrid) {
                newX = snapToGrid(
                  newX,
                  floorPlanData.gridSize ?? EDITOR_CONSTANTS.DEFAULT_GRID_SIZE
                );
                newY = snapToGrid(
                  newY,
                  floorPlanData.gridSize ?? EDITOR_CONSTANTS.DEFAULT_GRID_SIZE
                );
              }

              // 直接应用移动，不进行碰撞检测
              return { ...ws, x: newX, y: newY };
            }
            return ws;
          });

          // 拖动文本
          if (newData.texts) {
            newData.texts = prev.texts!.map((text) => {
              if (selectedIds.has(text.id)) {
                return {
                  ...text,
                  position: { x: text.position.x + dx, y: text.position.y + dy },
                };
              }
              return text;
            });
          }

          return newData;
        });

        // 更新上一次鼠标的画布坐标（使用 ref 避免状态更新延迟问题）
        lastElementMouseCanvasPos.current = point;
        return;
      }

      // 墙体绘制预览
      if (mode === "draw_wall" && wallDrawing.isDrawing) {
        // 检测是否接近已有的墙体节点
        const nearbyNode = findNearbyWallNode(point);
        const previewPoint = nearbyNode ? nearbyNode.point : point;
        wallDrawing.updatePreview(previewPoint, e.shiftKey);
      }
    },
    [
      isDragging,
      lastMousePos,
      isBoxSelecting,
      boxSelectStart,
      draggedElement,
      isDraggingPlanImage,
      mode,
      wallDrawing,
      getCanvasPoint,
      readOnly,
      selectedIds,
      findNearbyWallNode,
      floorPlanData,
      scale,
    ]
  );

  const handleCanvasMouseUp = useCallback(() => {
    const wasBoxSelecting = isBoxSelecting;

    // 清除拖动和画布平移状态
    setIsDragging(false);
    setDragStart(null);
    setLastMousePos(null);
    setDraggedElement(null);
    // 重置拖动相关的 ref
    lastElementMouseCanvasPos.current = null;
    dragStartCanvasPos.current = null;
    hasMovedBeyondThreshold.current = false;
    originalWorkstationPositions.current.clear();

    // 清除平面图拖拽状态
    setIsDraggingPlanImage(false);
    planImageDragStart.current = null;

    // 框选结束，选择框内的元素
    if (wasBoxSelecting && boxSelectStart && boxSelectEnd) {
      const minX = Math.min(boxSelectStart.x, boxSelectEnd.x);
      const maxX = Math.max(boxSelectStart.x, boxSelectEnd.x);
      const minY = Math.min(boxSelectStart.y, boxSelectEnd.y);
      const maxY = Math.max(boxSelectStart.y, boxSelectEnd.y);

      const newSelectedIds = new Set<string>();

      // 检查墙体 - 检查墙体线段与框选矩形是否相交
      if (!isLayerLocked("wall")) {
        for (const wall of floorPlanData.walls) {
          // 检查墙体是否与框选矩形相交（简单方法：检查墙体任意点是否在框内）
          for (const point of wall.points) {
            if (point.x >= minX && point.x <= maxX && point.y >= minY && point.y <= maxY) {
              newSelectedIds.add(wall.id);
              break;
            }
          }
        }
      }

      // 检查工位
      if (!isLayerLocked("workstation")) {
        for (const ws of floorPlanData.workstations) {
          // 检查工位中心是否在框内
          if (ws.x >= minX && ws.x <= maxX && ws.y >= minY && ws.y <= maxY) {
            newSelectedIds.add(ws.id);
          }
        }
      }

      // 检查门
      if (!isLayerLocked("door")) {
        for (const door of floorPlanData.doors) {
          const pos = door.position;
          if (pos.x >= minX && pos.x <= maxX && pos.y >= minY && pos.y <= maxY) {
            newSelectedIds.add(door.id);
          }
        }
      }

      // 检查文本
      if (!isLayerLocked("text")) {
        for (const text of floorPlanData.texts || []) {
          if (
            text.position.x >= minX &&
            text.position.x <= maxX &&
            text.position.y >= minY &&
            text.position.y <= maxY
          ) {
            newSelectedIds.add(text.id);
          }
        }
      }

      // 更新选择
      setSelectedIds(newSelectedIds);
      if (newSelectedIds.size > 0) {
        // 设置最后一个选中元素为当前选中
        const lastId = Array.from(newSelectedIds).pop()!;
        const elementType = getElementTypeById(lastId);
        if (elementType) {
          setSelectedId(lastId);
          setSelectedType(elementType);
        }
      }
    }

    setIsBoxSelecting(false);
    setBoxSelectStart(null);
    setBoxSelectEnd(null);

    // 拖动结束后，对选中的文本应用网格吸附（工位已在拖动过程中实时吸附）
    if (selectedIds.size > 0 && floorPlanData.snapToGrid) {
      setFloorPlanData((prev) => {
        const newData = { ...prev };

        // 吸附文本到网格
        if (newData.texts) {
          newData.texts = prev.texts!.map((text) => {
            if (selectedIds.has(text.id)) {
              const snappedX = snapToGrid(
                text.position.x,
                floorPlanData.gridSize ?? EDITOR_CONSTANTS.DEFAULT_GRID_SIZE
              );
              const snappedY = snapToGrid(
                text.position.y,
                floorPlanData.gridSize ?? EDITOR_CONSTANTS.DEFAULT_GRID_SIZE
              );
              return { ...text, position: { x: snappedX, y: snappedY } };
            }
            return text;
          });
        }

        return newData;
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- getElementTypeById and isLayerLocked are render-defined functions
  }, [isBoxSelecting, boxSelectStart, boxSelectEnd, floorPlanData, selectedIds]);

  const handleCanvasDoubleClick = useCallback(() => {
    if (mode === "draw_wall") {
      wallDrawing.finishDrawing();
    }
  }, [mode, wallDrawing]);

  // ==================== 键盘事件 ====================
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Delete" || e.key === "Backspace") {
        if ((selectedIds.size > 0 || selectedId) && !readOnly) {
          // 批量删除选中的元素
          if (selectedIds.size > 0) {
            let deletedCount = 0;
            setFloorPlanData((prev) => {
              const newData = { ...prev };

              // 删除选中的墙体
              if (!isLayerLocked("wall")) {
                const originalLength = newData.walls.length;
                newData.walls = newData.walls.filter((w) => !selectedIds.has(w.id));
                deletedCount += originalLength - newData.walls.length;
              }

              // 删除选中的门
              if (!isLayerLocked("door")) {
                const originalLength = newData.doors.length;
                newData.doors = newData.doors.filter((d) => !selectedIds.has(d.id));
                deletedCount += originalLength - newData.doors.length;
              }

              // 删除选中的工位
              if (!isLayerLocked("workstation")) {
                const originalLength = newData.workstations.length;
                newData.workstations = newData.workstations.filter((ws) => !selectedIds.has(ws.id));
                deletedCount += originalLength - newData.workstations.length;
              }

              // 删除选中的文本
              if (newData.texts && !isLayerLocked("text")) {
                const originalLength = newData.texts.length;
                newData.texts = newData.texts.filter((text) => !selectedIds.has(text.id));
                deletedCount += originalLength - newData.texts.length;
              }

              return newData;
            });
            if (deletedCount > 0) {
              message.success(`已删除 ${deletedCount} 个元素`);
            }
            handleDeselect();
          } else if (selectedId) {
            // 单个删除（兼容旧逻辑）
            if (!isLayerLocked(selectedType || "")) {
              if (selectedType === "wall") {
                setFloorPlanData((prev) => ({
                  ...prev,
                  walls: prev.walls.filter((w) => w.id !== selectedId),
                }));
              } else if (selectedType === "door") {
                setFloorPlanData((prev) => ({
                  ...prev,
                  doors: prev.doors.filter((d) => d.id !== selectedId),
                }));
              } else if (selectedType === "workstation") {
                setFloorPlanData((prev) => ({
                  ...prev,
                  workstations: prev.workstations.filter((ws) => ws.id !== selectedId),
                }));
              } else if (selectedType === "text") {
                setFloorPlanData((prev) => ({
                  ...prev,
                  texts: prev.texts?.filter((t) => t.id !== selectedId),
                }));
              }
              message.success("已删除");
              handleDeselect();
            }
          }
        }
      } else if (e.key === "Escape") {
        if (wallDrawing.isDrawing) {
          wallDrawing.cancelDrawing();
        } else {
          handleDeselect();
        }
      } else if (e.ctrlKey && e.key === "z") {
        e.preventDefault();
        handleUndo();
      } else if (e.ctrlKey && e.key === "y") {
        e.preventDefault();
        handleRedo();
      } else if (e.ctrlKey && e.key === "s") {
        e.preventDefault();
        handleSave();
      } else if (e.key === "Alt") {
        e.preventDefault();
        setIsAltPressed(true);
      }
    };

    const handleKeyUp = (e: KeyboardEvent) => {
      if (e.key === "Alt") {
        setIsAltPressed(false);
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    window.addEventListener("keyup", handleKeyUp);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
      window.removeEventListener("keyup", handleKeyUp);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- isLayerLocked/message/wallDrawing are render-defined or stable
  }, [
    selectedId,
    selectedType,
    selectedIds,
    readOnly,
    wallDrawing.isDrawing,
    handleDeselect,
    handleUndo,
    handleRedo,
    handleSave,
  ]);

  // ==================== 渲染 ====================
  return (
    <div
      className="cad-floor-plan-editor"
      style={{
        display: "flex",
        flexDirection: "column",
        height: "100%",
        background: CAD_COLOR_THEME.background,
        ...style,
      }}
    >
      {/* 工具栏 */}
      <CADToolbar
        mode={mode}
        onModeChange={setMode}
        onZoomIn={handleZoomIn}
        onZoomOut={handleZoomOut}
        onResetView={handleResetView}
        onSave={handleSave}
        canUndo={canUndo}
        canRedo={canRedo}
        onUndo={handleUndo}
        onRedo={handleRedo}
        readOnly={readOnly}
      />

      {/* 主内容区 */}
      <div style={{ display: "flex", flex: 1, overflow: "hidden" }}>
        {/* 左侧图层面板 */}
        <CADLayersPanel
          layers={layers}
          onLayerVisibilityChange={handleLayerVisibilityChange}
          onLayerLockChange={handleLayerLockChange}
          onLayerOpacityChange={handleLayerOpacityChange}
        />

        {/* 中间画布 */}
        <div
          style={{
            flex: 1,
            position: "relative",
            overflow: "hidden",
          }}
        >
          <svg
            ref={svgRef}
            width="100%"
            height="100%"
            onMouseDown={handleCanvasMouseDown}
            onMouseMove={handleCanvasMouseMove}
            onMouseUp={handleCanvasMouseUp}
            onDoubleClick={handleCanvasDoubleClick}
            onMouseLeave={handleCanvasMouseUp}
            style={{
              display: "block",
              cursor: isDragging
                ? "grabbing"
                : isAltPressed
                  ? "grab"
                  : ["draw_wall", "draw_door", "draw_workstation", "draw_text"].includes(mode)
                    ? "crosshair"
                    : "default",
            }}
          >
            <g transform={`translate(${offset.x}, ${offset.y}) scale(${scale})`}>
              {/* 网格背景 */}
              {isLayerVisible("grid") && (
                <>
                  <defs>
                    <pattern
                      id="grid"
                      width={floorPlanData.gridSize}
                      height={floorPlanData.gridSize}
                      patternUnits="userSpaceOnUse"
                    >
                      <path
                        d={`M ${floorPlanData.gridSize} 0 L 0 0 0 ${floorPlanData.gridSize}`}
                        fill="none"
                        stroke={CAD_COLOR_THEME.grid}
                        strokeWidth={0.5}
                      />
                    </pattern>
                  </defs>
                  <rect
                    width={floorPlanData.width}
                    height={floorPlanData.height}
                    fill="url(#grid)"
                  />
                </>
              )}

              {/* 平面图图层 */}
              {isLayerVisible("plan_image") && floorPlanData.planImageUrl && (
                <g
                  data-plan-image="true"
                  opacity={getLayerOpacity("plan_image") || 0.5}
                  transform={
                    floorPlanData.planImageTransform
                      ? `translate(${floorPlanData.planImageTransform.x}, ${floorPlanData.planImageTransform.y}) scale(${floorPlanData.planImageTransform.scale})`
                      : undefined
                  }
                  style={{
                    pointerEvents: readOnly || isLayerLocked("plan_image") ? "none" : "auto",
                    cursor:
                      readOnly || isLayerLocked("plan_image")
                        ? "default"
                        : isDraggingPlanImage
                          ? "grabbing"
                          : "grab",
                  }}
                  onMouseDown={(e) => {
                    if (e.button === 0 && !readOnly && !isLayerLocked("plan_image")) {
                      e.stopPropagation();
                      setIsDraggingPlanImage(true);
                      planImageDragStart.current = { x: e.clientX, y: e.clientY };
                    }
                  }}
                >
                  <image
                    href={floorPlanData.planImageUrl}
                    x={0}
                    y={0}
                    width={floorPlanData.width}
                    height={floorPlanData.height}
                    preserveAspectRatio="xMidYMid meet"
                  />
                </g>
              )}

              {/* 墙体 */}
              {isLayerVisible("wall") &&
                floorPlanData.walls.map((wall) => (
                  <WallElement
                    key={wall.id}
                    wall={wall}
                    selected={selectedIds.has(wall.id)}
                    hovered={hoveredId === wall.id}
                    onSelect={() => handleSelectElement(wall.id, "wall")}
                    onHover={(hovered) => setHoveredId(hovered ? wall.id : null)}
                  />
                ))}

              {/* 门 */}
              {isLayerVisible("door") &&
                floorPlanData.doors.map((door) => (
                  <DoorElement
                    key={door.id}
                    door={door}
                    selected={selectedIds.has(door.id)}
                    hovered={hoveredId === door.id}
                    onSelect={() => handleSelectElement(door.id, "door")}
                    onHover={(hovered) => setHoveredId(hovered ? door.id : null)}
                  />
                ))}

              {/* 工位 */}
              {isLayerVisible("workstation") &&
                floorPlanData.workstations.map((ws) => (
                  <WorkstationElement
                    key={ws.id}
                    workstation={ws}
                    selected={selectedIds.has(ws.id)}
                    hovered={hoveredId === ws.id}
                    onSelect={() => handleSelectElement(ws.id, "workstation")}
                    onHover={(hovered) => setHoveredId(hovered ? ws.id : null)}
                  />
                ))}

              {/* 文本 */}
              {isLayerVisible("text") &&
                (floorPlanData.texts || []).map((text) => (
                  <CADTextElement
                    key={text.id}
                    text={text}
                    selected={selectedIds.has(text.id)}
                    hovered={hoveredId === text.id}
                    onSelect={() => handleSelectElement(text.id, "text")}
                    onHover={(hovered) => setHoveredId(hovered ? text.id : null)}
                  />
                ))}

              {/* 墙体绘制预览 */}
              {wallDrawing.isDrawing && wallDrawing.previewPoint && (
                <>
                  {wallDrawing.drawPoints.length > 0 && (
                    <path
                      d={wallDrawing.drawPoints
                        .map((p, i) => `${i === 0 ? "M" : "L"} ${p.x} ${p.y}`)
                        .join(" ")}
                      stroke="#337ab0"
                      strokeWidth={2}
                      fill="none"
                      strokeDasharray="5,5"
                    />
                  )}
                  {wallDrawing.drawPoints.length > 0 && (
                    <line
                      x1={wallDrawing.drawPoints[wallDrawing.drawPoints.length - 1].x}
                      y1={wallDrawing.drawPoints[wallDrawing.drawPoints.length - 1].y}
                      x2={wallDrawing.previewPoint.x}
                      y2={wallDrawing.previewPoint.y}
                      stroke="#337ab0"
                      strokeWidth={2}
                      strokeDasharray="5,5"
                    />
                  )}
                  <circle
                    cx={wallDrawing.previewPoint.x}
                    cy={wallDrawing.previewPoint.y}
                    r={4}
                    fill="#337ab0"
                  />
                </>
              )}

              {/* 框选矩形 */}
              {isBoxSelecting && boxSelectStart && boxSelectEnd && (
                <rect
                  x={Math.min(boxSelectStart.x, boxSelectEnd.x)}
                  y={Math.min(boxSelectStart.y, boxSelectEnd.y)}
                  width={Math.abs(boxSelectEnd.x - boxSelectStart.x)}
                  height={Math.abs(boxSelectEnd.y - boxSelectStart.y)}
                  fill="rgba(51, 122, 176, 0.1)"
                  stroke="#337ab0"
                  strokeWidth={1}
                  strokeDasharray="4,4"
                  pointerEvents="none"
                />
              )}
            </g>
          </svg>
        </div>

        {/* 右侧属性面板 */}
        <CADPropertyPanel
          selectedElement={selectedElement}
          onUpdate={handleUpdateElement}
          readOnly={readOnly}
        />
      </div>

      {/* 文本输入对话框 */}
      <Modal
        title="输入文本"
        open={isTextInputVisible}
        onOk={handleTextInputOk}
        onCancel={handleTextInputCancel}
        okText="确定"
        cancelText="取消"
      >
        <Input.TextArea
          value={tempTextContent}
          onChange={(e) => setTempTextContent(e.target.value)}
          placeholder="请输入文本内容..."
          autoFocus
          rows={4}
          maxLength={200}
        />
      </Modal>
    </div>
  );
}
