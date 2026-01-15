import React, { useState, useRef, useCallback, useEffect } from 'react';
import { Card, Button, Dropdown, Tooltip, message } from 'antd';
import {
  ZoomInOutlined,
  ZoomOutOutlined,
  CompressOutlined,
  ReloadOutlined,
  AppstoreOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons';
import { getWorkstationStatusColor, getWorkstationTypeColor } from '@/utils/three/colors';
import './FloorPlanEditor.less';

/**
 * 工位节点接口
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

/**
 * FloorPlanEditor 组件属性
 */
export interface FloorPlanEditorProps {
  floorId: string;
  workstations: WorkstationNode[];
  onUpdatePosition: (items: {id: string; positionX: number; positionY: number; rotation?: number}[]) => Promise<void>;
  onEdit: (workstation: WorkstationNode) => void;
}

/**
 * 视图状态接口
 */
interface ViewState {
  scale: number;
  offsetX: number;
  offsetY: number;
  isDragging: boolean;
  dragStartX: number;
  dragStartY: number;
}

/**
 * 拖拽状态接口
 */
interface DragState {
  isDragging: boolean;
  workstationId: string | null;
  startX: number;
  startY: number;
  originalX: number;
  originalY: number;
}

const GRID_SIZE = 20;
const DEFAULT_SCALE = 1;
const MIN_SCALE = 0.25;
const MAX_SCALE = 3;

/**
 * 获取工位颜色
 * 优先显示状态颜色（空闲/占用），次要显示类型颜色
 */
const getWorkstationColor = (status: number): { main: string; border: string; bg: string } => {
  // status: 0=空闲, 1=占用, 2=维护
  const statusColor = getWorkstationStatusColor(status);

  return {
    main: statusColor.main,
    border: statusColor.border,
    bg: statusColor.bg,
  };
};

/**
 * FloorPlanEditor 组件
 */
const FloorPlanEditor: React.FC<FloorPlanEditorProps> = ({
  floorId: _floorId,
  workstations,
  onUpdatePosition,
  onEdit,
}) => {
  const containerRef = useRef<SVGSVGElement>(null);
  const cardRef = useRef<HTMLDivElement>(null);
  const [containerSize, setContainerSize] = useState({ width: 800, height: 600 });

  // 监听容器尺寸变化
  useEffect(() => {
    const cardElement = cardRef.current;
    if (!cardElement) return;

    const updateSize = () => {
      const rect = cardElement.getBoundingClientRect();
      const toolbarHeight = 48; // 工具栏高度
      setContainerSize({
        width: rect.width,
        height: rect.height - toolbarHeight
      });
    };

    // 初始尺寸
    updateSize();

    // 使用 ResizeObserver 监听尺寸变化
    const resizeObserver = new ResizeObserver(updateSize);
    resizeObserver.observe(cardElement);

    return () => {
      resizeObserver.disconnect();
    };
  }, []);

  const width = containerSize.width;
  const height = containerSize.height;
  const [viewState, setViewState] = useState<ViewState>({
    scale: DEFAULT_SCALE,
    offsetX: 0,
    offsetY: 0,
    isDragging: false,
    dragStartX: 0,
    dragStartY: 0
  });

  // 拖拽状态
  const [dragState, setDragState] = useState<DragState>({
    isDragging: false,
    workstationId: null,
    startX: 0,
    startY: 0,
    originalX: 0,
    originalY: 0
  });

  // 当前拖拽的工位临时位置
  const [draggedNodePos, setDraggedNodePos] = useState<{x: number; y: number} | null>(null);

  // 选中状态
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [hoveredId, setHoveredId] = useState<string | null>(null);
  const [showGrid, setShowGrid] = useState(true);
  const [contextMenu, setContextMenu] = useState<{
    visible: boolean;
    x: number;
    y: number;
    workstation: WorkstationNode | null;
  }>({
    visible: false,
    x: 0,
    y: 0,
    workstation: null
  });

  /**
   * 屏幕坐标转换为 SVG 坐标
   */
  const screenToSvg = useCallback((x: number, y: number) => {
    const rect = containerRef.current?.getBoundingClientRect();
    if (!rect) return { x: 0, y: 0 };

    return {
      x: (x - rect.left - viewState.offsetX) / viewState.scale,
      y: (y - rect.top - viewState.offsetY) / viewState.scale
    };
  }, [viewState.offsetX, viewState.offsetY, viewState.scale]);

  /**
   * 处理鼠标按下 - 开始拖拽工位或平移画布
   */
  const handleMouseDown = useCallback((e: React.MouseEvent<SVGSVGElement>) => {
    // 检查是否点击了工位
    const target = e.target as SVGElement;
    const workstationEl = target.closest('[data-workstation-id]');
    const spacePressed = (e.nativeEvent as any).code === 'Space';

    if (workstationEl && !spacePressed) {
      // 点击了工位
      const workstationId = workstationEl.getAttribute('data-workstation-id');
      const workstation = workstations.find(w => w.id === workstationId);

      if (workstation && workstation.status === 0) {
        setDragState({
          isDragging: true,
          workstationId: workstationId,
          startX: e.clientX,
          startY: e.clientY,
          originalX: workstation.x,
          originalY: workstation.y
        });

        setSelectedId(workstationId);
      }
    } else if (spacePressed || !workstationEl) {
      // 点击背景或按住空格 - 平移画布
      setViewState(prev => ({
        ...prev,
        isDragging: true,
        dragStartX: e.clientX - prev.offsetX,
        dragStartY: e.clientY - prev.offsetY
      }));

      if (!workstationEl) {
        setSelectedId(null);
      }
    }

    // 隐藏右键菜单
    setContextMenu(prev => ({ ...prev, visible: false }));
  }, [workstations, screenToSvg]);

  /**
   * 处理鼠标移动 - 拖拽工位或平移画布
   */
  const handleMouseMove = useCallback((e: globalThis.MouseEvent) => {
    if (dragState.isDragging && dragState.workstationId) {
      // 拖拽工位 - 更新临时位置，不修改props
      const dx = (e.clientX - dragState.startX) / viewState.scale;
      const dy = (e.clientY - dragState.startY) / viewState.scale;

      setDraggedNodePos({
        x: dragState.originalX + dx,
        y: dragState.originalY + dy
      });
    } else if (viewState.isDragging) {
      // 平移画布
      setViewState(prev => ({
        ...prev,
        offsetX: e.clientX - prev.dragStartX,
        offsetY: e.clientY - prev.dragStartY
      }));
    }
  }, [dragState, viewState.scale, viewState.isDragging, viewState.dragStartX, viewState.dragStartY]);

  /**
   * 处理鼠标松开 - 结束拖拽
   */
  const handleMouseUp = useCallback(async () => {
    if (dragState.isDragging && dragState.workstationId && draggedNodePos) {
      // 拖拽结束，保存位置
      const workstation = workstations.find(w => w.id === dragState.workstationId);
      if (workstation) {
        // 网格吸附 - 使用临时位置
        const snappedX = Math.round(draggedNodePos.x / GRID_SIZE) * GRID_SIZE;
        const snappedY = Math.round(draggedNodePos.y / GRID_SIZE) * GRID_SIZE;

        await onUpdatePosition([{
          id: workstation.id,
          positionX: snappedX,
          positionY: snappedY
        }]);

        message.success('工位位置已更新');
      }

      // 清除临时拖拽位置
      setDraggedNodePos(null);
    }

    setDragState({
      isDragging: false,
      workstationId: null,
      startX: 0,
      startY: 0,
      originalX: 0,
      originalY: 0
    });

    setViewState(prev => ({ ...prev, isDragging: false }));
  }, [dragState, draggedNodePos, workstations, onUpdatePosition]);

  /**
   * 处理滚轮 - 缩放画布（使用原生事件以避免 passive 警告）
   */
  const handleWheel = useCallback((e: WheelEvent) => {
    // Ctrl + 滚轮缩放
    const isCtrlPressed = e.ctrlKey || e.metaKey;

    if (isCtrlPressed) {
      e.preventDefault();
      const delta = e.deltaY > 0 ? -0.1 : 0.1;
      const newScale = Math.min(MAX_SCALE, Math.max(MIN_SCALE, viewState.scale + delta));

      // 以鼠标位置为中心缩放
      const rect = containerRef.current?.getBoundingClientRect();
      if (!rect) return;

      const mouseX = e.clientX - rect.left;
      const mouseY = e.clientY - rect.top;

      const scaleChange = newScale / viewState.scale;
      const newOffsetX = mouseX - (mouseX - viewState.offsetX) * scaleChange;
      const newOffsetY = mouseY - (mouseY - viewState.offsetY) * scaleChange;

      setViewState(prev => ({
        ...prev,
        scale: newScale,
        offsetX: newOffsetX,
        offsetY: newOffsetY
      }));
    } else {
      // 普通滚轮 - 平移
      e.preventDefault();
      setViewState(prev => ({
        ...prev,
        offsetX: prev.offsetX - e.deltaX,
        offsetY: prev.offsetY - e.deltaY
      }));
    }
  }, [viewState.scale, viewState.offsetX, viewState.offsetY]);

  // 添加非 passive 的 wheel 事件监听器
  useEffect(() => {
    const svgElement = containerRef.current;
    if (!svgElement) return;

    svgElement.addEventListener('wheel', handleWheel, { passive: false });

    return () => {
      svgElement.removeEventListener('wheel', handleWheel);
    };
  }, [handleWheel]);

  /**
   * 缩放画布
   */
  const zoom = useCallback((delta: number) => {
    const newScale = Math.min(MAX_SCALE, Math.max(MIN_SCALE, viewState.scale + delta));
    const centerX = width / 2;
    const centerY = height / 2;

    const scaleChange = newScale / viewState.scale;
    const newOffsetX = centerX - (centerX - viewState.offsetX) * scaleChange;
    const newOffsetY = centerY - (centerY - viewState.offsetY) * scaleChange;

    setViewState(prev => ({
      ...prev,
      scale: newScale,
      offsetX: newOffsetX,
      offsetY: newOffsetY
    }));
  }, [viewState.scale, viewState.offsetX, viewState.offsetY, width, height]);

  /**
   * 重置视图
   */
  const resetView = useCallback(() => {
    setViewState({
      scale: DEFAULT_SCALE,
      offsetX: 0,
      offsetY: 0,
      isDragging: false,
      dragStartX: 0,
      dragStartY: 0
    });
  }, []);

  /**
   * 适应视图
   */
  const fitToView = useCallback(() => {
    if (workstations.length === 0) return;

    // 计算边界
    let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;

    workstations.forEach(w => {
      minX = Math.min(minX, w.x);
      minY = Math.min(minY, w.y);
      maxX = Math.max(maxX, w.x + w.width);
      maxY = Math.max(maxY, w.y + w.height);
    });

    const contentWidth = maxX - minX + 100;
    const contentHeight = maxY - minY + 100;

    const scaleX = (width - 100) / contentWidth;
    const scaleY = (height - 100) / contentHeight;
    const newScale = Math.min(scaleX, scaleY, MAX_SCALE);

    const centerX = (minX + maxX) / 2;
    const centerY = (minY + maxY) / 2;

    const offsetX = width / 2 - centerX * newScale;
    const offsetY = height / 2 - centerY * newScale;

    setViewState({
      scale: newScale,
      offsetX,
      offsetY,
      isDragging: false,
      dragStartX: 0,
      dragStartY: 0
    });
  }, [workstations, width, height]);

  /**
   * 处理工位点击
   */
  const handleWorkstationClick = useCallback((workstation: WorkstationNode, e: React.MouseEvent) => {
    e.stopPropagation();
    setSelectedId(workstation.id);
  }, []);

  /**
   * 处理工位双击
   */
  const handleWorkstationDoubleClick = useCallback((workstation: WorkstationNode, e: React.MouseEvent) => {
    e.stopPropagation();
    onEdit(workstation);
  }, [onEdit]);

  /**
   * 处理右键菜单
   */
  const handleContextMenu = useCallback((workstation: WorkstationNode, e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();

    setContextMenu({
      visible: true,
      x: e.clientX,
      y: e.clientY,
      workstation
    });
  }, []);

  /**
   * 菜单项操作
   */
  const handleMenuAction = useCallback(async (action: string) => {
    if (!contextMenu.workstation) return;

    switch (action) {
      case 'edit':
        onEdit(contextMenu.workstation);
        break;
      case 'rotate-45':
        await onUpdatePosition([{
          id: contextMenu.workstation.id,
          positionX: contextMenu.workstation.x,
          positionY: contextMenu.workstation.y,
          rotation: ((contextMenu.workstation.rotation || 0) + 45) % 360
        }]);
        message.success('工位已顺时针旋转45°');
        break;
      case 'rotate-90':
        await onUpdatePosition([{
          id: contextMenu.workstation.id,
          positionX: contextMenu.workstation.x,
          positionY: contextMenu.workstation.y,
          rotation: ((contextMenu.workstation.rotation || 0) + 90) % 360
        }]);
        message.success('工位已顺时针旋转90°');
        break;
      case 'rotate-180':
        await onUpdatePosition([{
          id: contextMenu.workstation.id,
          positionX: contextMenu.workstation.x,
          positionY: contextMenu.workstation.y,
          rotation: ((contextMenu.workstation.rotation || 0) + 180) % 360
        }]);
        message.success('工位已旋转180°');
        break;
      case 'rotate-reset':
        await onUpdatePosition([{
          id: contextMenu.workstation.id,
          positionX: contextMenu.workstation.x,
          positionY: contextMenu.workstation.y,
          rotation: 0
        }]);
        message.success('工位方向已重置');
        break;
      case 'delete':
        message.warning('删除功能待实现');
        break;
      case 'copy':
        message.warning('复制功能待实现');
        break;
    }

    setContextMenu(prev => ({ ...prev, visible: false }));
  }, [contextMenu.workstation, onEdit, onUpdatePosition]);

  // 注册全局鼠标事件
  useEffect(() => {
    if (dragState.isDragging || viewState.isDragging) {
      window.addEventListener('mousemove', handleMouseMove);
      window.addEventListener('mouseup', handleMouseUp);

      return () => {
        window.removeEventListener('mousemove', handleMouseMove);
        window.removeEventListener('mouseup', handleMouseUp);
      };
    }
  }, [dragState.isDragging, viewState.isDragging, handleMouseMove, handleMouseUp]);

  // 键盘事件
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === ' ') {
        e.preventDefault();
        (containerRef.current?.style as any).cursor = 'grab';
      }
    };

    const handleKeyUp = (e: KeyboardEvent) => {
      if (e.key === ' ') {
        (containerRef.current?.style as any).cursor = 'default';
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    window.addEventListener('keyup', handleKeyUp);

    return () => {
      window.removeEventListener('keydown', handleKeyDown);
      window.removeEventListener('keyup', handleKeyUp);
    };
  }, []);

  /**
   * 渲染网格
   */
  const renderGrid = () => {
    if (!showGrid) return null;

    const gridSize = GRID_SIZE * viewState.scale;
    const offsetX = viewState.offsetX % gridSize;
    const offsetY = viewState.offsetY % gridSize;

    const verticalLines = [];
    const horizontalLines = [];

    for (let x = offsetX; x < width; x += gridSize) {
      verticalLines.push(<line key={`v-${x}`} x1={x} y1={0} x2={x} y2={height} stroke="#e8e8e8" strokeWidth="1" />);
    }

    for (let y = offsetY; y < height; y += gridSize) {
      horizontalLines.push(<line key={`h-${y}`} x1={0} y1={y} x2={width} y2={y} stroke="#e8e8e8" strokeWidth="1" />);
    }

    return (
      <g className="floor-plan-grid">
        {verticalLines}
        {horizontalLines}
      </g>
    );
  };

  /**
   * 渲染工位
   */
  const renderWorkstations = () => {
    return workstations.map(workstation => {
      const colors = getWorkstationColor(workstation.status);
      const statusColor = getWorkstationStatusColor(workstation.status);
      const typeColor = getWorkstationTypeColor(workstation.type);
      const isSelected = selectedId === workstation.id;
      const isHovered = hoveredId === workstation.id;
      const isDragging = dragState.workstationId === workstation.id;

      // 位置和尺寸 - 拖拽时使用临时位置
      const posX = isDragging && draggedNodePos ? draggedNodePos.x : workstation.x;
      const posY = isDragging && draggedNodePos ? draggedNodePos.y : workstation.y;

      // 旋转角度（默认0）
      const rotation = workstation.rotation || 0;

      const x = posX * viewState.scale + viewState.offsetX;
      const y = posY * viewState.scale + viewState.offsetY;
      const w = workstation.width * viewState.scale;
      const h = workstation.height * viewState.scale;

      // 样式 - 只使用 filter，不使用 transform
      const style: React.CSSProperties = {
        cursor: workstation.status === 0 ? 'move' : 'not-allowed',
        filter: isHovered && workstation.status === 0 ? 'drop-shadow(0 4px 8px rgba(0,0,0,0.15))' : undefined
      };

      // 悬浮时的缩放效果（SVG transform，不影响位置）
      const hoverTransform = isHovered && workstation.status === 0
        ? `translate(${w / 2}, ${h / 2}) scale(1.05) translate(${-w / 2}, ${-h / 2})`
        : '';

      // 应用旋转：围绕工位中心旋转
      const rotationTransform = rotation !== 0
        ? `rotate(${rotation}, ${w / 2}, ${h / 2})`
        : '';

      // 组合变换：先悬浮效果，再旋转
      const combinedTransform = rotationTransform
        ? `${rotationTransform} ${hoverTransform}`.trim()
        : hoverTransform;

      return (
        <g key={workstation.id} transform={`translate(${x}, ${y})`}>
          <g
            data-workstation-id={workstation.id}
            transform={combinedTransform}
            onMouseEnter={() => setHoveredId(workstation.id)}
            onMouseLeave={() => setHoveredId(null)}
            onClick={(e) => handleWorkstationClick(workstation, e)}
            onDoubleClick={(e) => handleWorkstationDoubleClick(workstation, e)}
            onContextMenu={(e) => handleContextMenu(workstation, e)}
            style={style}
          >
            {/* 工位桌面 - 带圆角的矩形 */}
            <rect
              width={w}
              height={h}
              fill={colors.main}
              stroke={isSelected ? '#ff4d4f' : colors.border}
              strokeWidth={isSelected ? 3 : 2}
              rx={6}
              ry={6}
              opacity={isDragging ? 0.8 : 1}
            />

            {/* 桌面高光效果 - 增加立体感 */}
            <rect
              x={2}
              y={2}
              width={w - 4}
              height={h * 0.3}
              fill="url(#deskHighlight)"
              rx={4}
              ry={4}
              opacity={0.3}
              pointerEvents="none"
            />

            {/* 显示器 - 位于桌面上方 */}
            <rect
              x={w * 0.25}
              y={h * 0.1}
              width={w * 0.5}
              height={h * 0.35}
              fill="#1a1a1a"
              stroke="#2d2d2d"
              strokeWidth={1.5}
              rx={2}
              ry={2}
              opacity={0.9}
              pointerEvents="none"
            />
            {/* 显示器屏幕 - 选中或悬停时发光 */}
            <rect
              x={w * 0.27}
              y={h * 0.12}
              width={w * 0.46}
              height={h * 0.31}
              fill={isHovered || isSelected ? statusColor.main : '#0a0a0a'}
              opacity={isHovered || isSelected ? 0.6 : 0.8}
              rx={1}
              ry={1}
              pointerEvents="none"
            />
            {/* 显示器支架 */}
            <rect
              x={w * 0.48}
              y={h * 0.45}
              width={w * 0.04}
              height={h * 0.08}
              fill="#333333"
              opacity={0.8}
              pointerEvents="none"
            />
            {/* 显示器底座 */}
            <rect
              x={w * 0.43}
              y={h * 0.52}
              width={w * 0.14}
              height={h * 0.02}
              fill="#2d2d2d"
              opacity={0.8}
              pointerEvents="none"
            />

            {/* 键盘 */}
            <rect
              x={w * 0.15}
              y={h * 0.58}
              width={w * 0.35}
              height={h * 0.12}
              fill="#2d2d2d"
              stroke="#1a1a1a"
              strokeWidth={0.5}
              rx={1}
              ry={1}
              opacity={0.85}
              pointerEvents="none"
            />
            {/* 键盘按键细节 */}
            <g opacity={0.3} pointerEvents="none">
              <rect x={w * 0.16} y={h * 0.59} width={w * 0.08} height={h * 0.03} fill="#1a1a1a" rx={0.5} />
              <rect x={w * 0.25} y={h * 0.59} width={w * 0.08} height={h * 0.03} fill="#1a1a1a" rx={0.5} />
              <rect x={w * 0.34} y={h * 0.59} width={w * 0.08} height={h * 0.03} fill="#1a1a1a" rx={0.5} />
              <rect x={w * 0.16} y={h * 0.63} width={w * 0.08} height={h * 0.03} fill="#1a1a1a" rx={0.5} />
              <rect x={w * 0.25} y={h * 0.63} width={w * 0.08} height={h * 0.03} fill="#1a1a1a" rx={0.5} />
              <rect x={w * 0.34} y={h * 0.63} width={w * 0.08} height={h * 0.03} fill="#1a1a1a" rx={0.5} />
            </g>

            {/* 鼠标垫 */}
            <ellipse
              cx={w * 0.7}
              cy={h * 0.62}
              rx={w * 0.08}
              ry={h * 0.06}
              fill="#4a4a4a"
              opacity={0.7}
              pointerEvents="none"
            />

            {/* 椅子 - 在桌面前方 */}
            <g opacity={0.85} pointerEvents="none">
              {/* 椅子靠背 */}
              <rect
                x={w * 0.35}
                y={h * 0.75}
                width={w * 0.3}
                height={h * 0.12}
                fill={statusColor.main}
                stroke={statusColor.border}
                strokeWidth={1}
                rx={2}
                ry={2}
              />
              {/* 椅子座位 */}
              <rect
                x={w * 0.35}
                y={h * 0.87}
                width={w * 0.3}
                height={h * 0.08}
                fill="#3d3d3d"
                stroke="#2d2d2d"
                strokeWidth={1}
                rx={2}
                ry={2}
              />
              {/* 椅子扶手 */}
              <rect
                x={w * 0.28}
                y={h * 0.82}
                width={w * 0.06}
                height={h * 0.06}
                fill="#2d2d2d"
                rx={1}
                ry={1}
              />
              <rect
                x={w * 0.66}
                y={h * 0.82}
                width={w * 0.06}
                height={h * 0.06}
                fill="#2d2d2d"
                rx={1}
                ry={1}
              />
            </g>

            {/* 工位编号 */}
            <text
              x={w / 2}
              y={h * 0.3}
              textAnchor="middle"
              fill="#ffffff"
              fontSize={Math.max(10, 12 * viewState.scale)}
              fontWeight="bold"
              pointerEvents="none"
              opacity={0.95}
            >
              {workstation.code}
            </text>

            {/* 工位状态标签 */}
            <text
              x={w / 2}
              y={h * 0.42}
              textAnchor="middle"
              fill="#ffffff"
              fontSize={Math.max(7, 8 * viewState.scale)}
              pointerEvents="none"
              opacity={0.9}
            >
              {statusColor.label}
            </text>

            {/* 工位类型小点 - 右上角 */}
            <circle
              cx={w - 8}
              cy={8}
              r={4}
              fill={typeColor.main}
              opacity={0.8}
              pointerEvents="none"
            />
          </g>
        </g>
      );
    });
  };

  /**
   * 渲染右键菜单
   */
  const renderContextMenu = () => {
    if (!contextMenu.visible || !contextMenu.workstation) return null;

    const menuItems = [
      { key: 'edit', label: '编辑工位', icon: <InfoCircleOutlined /> },
      { type: 'divider' },
      { key: 'rotate-45', label: '顺时针旋转45°', icon: <InfoCircleOutlined /> },
      { key: 'rotate-90', label: '顺时针旋转90°', icon: <InfoCircleOutlined /> },
      { key: 'rotate-180', label: '旋转180°', icon: <InfoCircleOutlined /> },
      { key: 'rotate-reset', label: '重置方向', icon: <InfoCircleOutlined /> },
      { type: 'divider' },
      { key: 'copy', label: '复制工位', icon: <InfoCircleOutlined /> },
      { key: 'delete', label: '删除工位', icon: <InfoCircleOutlined />, danger: true }
    ];

    return (
      <Dropdown
        menu={{
          items: menuItems as any,
          onClick: ({ key }) => handleMenuAction(key)
        }}
        open={contextMenu.visible}
        onOpenChange={(visible) => setContextMenu(prev => ({ ...prev, visible }))}
        trigger={[]}
      >
        <div
          style={{
            position: 'fixed',
            left: contextMenu.x,
            top: contextMenu.y,
            width: 1,
            height: 1,
            pointerEvents: 'none'
          }}
        />
      </Dropdown>
    );
  };

  // 获取选中的工位
  const selectedWorkstation = workstations.find(w => w.id === selectedId);

  return (
    <Card
      ref={cardRef}
      className="floor-plan-editor"
      styles={{ body: { padding: 0 } }}
      style={{ height: '100%', overflow: 'hidden' }}
    >
      {/* 工具栏 */}
      <div className="floor-plan-toolbar" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '8px 12px', borderBottom: '1px solid #f0f0f0' }}>
        <div className="toolbar-left">
          <Tooltip title="放大 (Ctrl+滚轮)">
            <Button
              type="text"
              icon={<ZoomInOutlined />}
              onClick={() => zoom(0.1)}
              disabled={viewState.scale >= MAX_SCALE}
            />
          </Tooltip>
          <Tooltip title="缩小 (Ctrl+滚轮)">
            <Button
              type="text"
              icon={<ZoomOutOutlined />}
              onClick={() => zoom(-0.1)}
              disabled={viewState.scale <= MIN_SCALE}
            />
          </Tooltip>
          <span className="zoom-level">{Math.round(viewState.scale * 100)}%</span>
          <Tooltip title="适应视图">
            <Button
              type="text"
              icon={<CompressOutlined />}
              onClick={fitToView}
            />
          </Tooltip>
          <Tooltip title="重置视图">
            <Button
              type="text"
              icon={<ReloadOutlined />}
              onClick={resetView}
            />
          </Tooltip>
          <Tooltip title={showGrid ? '隐藏网格' : '显示网格'}>
            <Button
              type="text"
              icon={<AppstoreOutlined />}
              onClick={() => setShowGrid(!showGrid)}
            />
          </Tooltip>
        </div>
        <div className="toolbar-right">
          {selectedWorkstation && (
            <span className="selected-info">
              <InfoCircleOutlined /> 已选择: {selectedWorkstation.code} - {selectedWorkstation.name}
              ({Math.round(selectedWorkstation.x)}, {Math.round(selectedWorkstation.y)})
            </span>
          )}
        </div>
      </div>

      {/* SVG 画布 */}
      <svg
        ref={containerRef}
        className="floor-plan-canvas"
        width={width}
        height={height - 48}
        onMouseDown={handleMouseDown}
        style={{
          cursor: viewState.isDragging ? 'grabbing' : 'default',
          background: '#f5f5f5',
          display: 'block'
        }}
      >
        {/* SVG 定义 - 渐变和滤镜 */}
        <defs>
          {/* 桌面高光渐变 */}
          <linearGradient id="deskHighlight" x1="0%" y1="0%" x2="0%" y2="100%">
            <stop offset="0%" stopColor="#ffffff" stopOpacity={0.4} />
            <stop offset="100%" stopColor="#ffffff" stopOpacity={0} />
          </linearGradient>
        </defs>

        {/* 背景填充 */}
        <rect width={width} height={height} fill="#fafafa" />

        {/* 网格 */}
        {renderGrid()}

        {/* 工位 */}
        {renderWorkstations()}

        {/* 拖拽时的坐标提示 */}
        {dragState.isDragging && dragState.workstationId && (
          <g>
            <rect
              x={width / 2 - 100}
              y={16}
              width={200}
              height={32}
              fill="rgba(0, 0, 0, 0.75)"
              rx={4}
            />
            <text
              x={width / 2}
              y={37}
              textAnchor="middle"
              fill="#fff"
              fontSize={14}
            >
              X: {Math.round(dragState.originalX)}, Y: {Math.round(dragState.originalY)}
            </text>
          </g>
        )}
      </svg>

      {/* 右键菜单 */}
      {renderContextMenu()}
    </Card>
  );
};

export default FloorPlanEditor;
