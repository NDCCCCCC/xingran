/**
 * FloorPlanEditor 平面图编辑器主组件
 */

import React, { useState, useRef, useCallback, useEffect } from "react";
import type { CSSProperties } from "react";
import { Card, Button, Dropdown, Tooltip, App } from "antd";
import type { MenuProps } from "antd";
import {
  ZoomInOutlined,
  ZoomOutOutlined,
  CompressOutlined,
  ReloadOutlined,
  AppstoreOutlined,
  InfoCircleOutlined,
} from "@ant-design/icons";
import { getWorkstationStatusColor, getWorkstationTypeColor } from "@/utils/three/colors";
import "./FloorPlanEditor.less";

// 导入提取的类型、常量和 Hooks
import type { WorkstationNode, FloorPlanEditorProps } from "./FloorPlanEditor.types";
import { GRID_SIZE, TOOLBAR_HEIGHT } from "./FloorPlanEditor.constants";
import { usePanZoom } from "./FloorPlanEditor.panZoom";
import { useWorkstationDrag } from "./FloorPlanEditor.hooks";

/**
 * 获取工位颜色
 */
const getWorkstationColor = (status: number): { main: string; border: string; bg: string } => {
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
  const { message } = App.useApp();
  const containerRef = useRef<SVGSVGElement>(null);
  const cardRef = useRef<HTMLDivElement>(null);
  const [containerSize, setContainerSize] = useState({ width: 800, height: 600 });

  // 使用 PanZoom Hook
  const {
    viewState,
    zoomIn,
    zoomOut,
    resetView,
    fitToScreen,
    handlePanStart,
    handlePanMove,
    handlePanEnd,
    handleWheel,
    screenToSvg: _screenToSvg,
  } = usePanZoom({ containerRef });

  // 使用 WorkstationDrag Hook
  const {
    dragState,
    draggedNodePos,
    handleStartDrag,
    handleDragMove,
    handleEndDrag,
    clearDraggedPos,
  } = useWorkstationDrag({ workstations, onUpdatePosition });

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
    workstation: null,
  });

  // 监听容器尺寸变化
  useEffect(() => {
    const cardElement = cardRef.current;
    if (!cardElement) return;

    const updateSize = () => {
      const rect = cardElement.getBoundingClientRect();
      setContainerSize({
        width: rect.width,
        height: rect.height - TOOLBAR_HEIGHT,
      });
    };

    updateSize();
    const resizeObserver = new ResizeObserver(updateSize);
    resizeObserver.observe(cardElement);

    return () => {
      resizeObserver.disconnect();
    };
  }, []);

  // 添加非 passive 的 wheel 事件监听器
  useEffect(() => {
    const svgElement = containerRef.current;
    if (!svgElement) return;

    svgElement.addEventListener("wheel", handleWheel, { passive: false });

    return () => {
      svgElement.removeEventListener("wheel", handleWheel);
    };
  }, [handleWheel]);

  // 注册全局鼠标事件
  useEffect(() => {
    if (dragState.isDragging || viewState.isDragging) {
      const handleMouseMove = (e: globalThis.MouseEvent) => {
        if (dragState.isDragging && dragState.workstationId) {
          handleDragMove(e.clientX, e.clientY, viewState.scale);
        } else if (viewState.isDragging) {
          handlePanMove(e.clientX, e.clientY);
        }
      };

      const handleMouseUp = async () => {
        if (dragState.isDragging && dragState.workstationId && draggedNodePos) {
          await handleEndDrag();
        }
        handlePanEnd();
        clearDraggedPos();
      };

      window.addEventListener("mousemove", handleMouseMove);
      window.addEventListener("mouseup", handleMouseUp);

      return () => {
        window.removeEventListener("mousemove", handleMouseMove);
        window.removeEventListener("mouseup", handleMouseUp);
      };
    }
  }, [
    dragState,
    draggedNodePos,
    viewState,
    handleDragMove,
    handlePanMove,
    handleEndDrag,
    handlePanEnd,
    clearDraggedPos,
  ]);

  // 键盘事件
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === " ") {
        e.preventDefault();
        if (containerRef.current?.style) {
          (containerRef.current.style as CSSProperties).cursor = "grab";
        }
      }
    };

    const handleKeyUp = (e: KeyboardEvent) => {
      if (e.key === " ") {
        if (containerRef.current?.style) {
          (containerRef.current.style as CSSProperties).cursor = "default";
        }
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    window.addEventListener("keyup", handleKeyUp);

    return () => {
      window.removeEventListener("keydown", handleKeyDown);
      window.removeEventListener("keyup", handleKeyUp);
    };
  }, []);

  /**
   * 处理鼠标按下 - 开始拖拽工位或平移画布
   */
  const handleMouseDown = useCallback(
    (e: React.MouseEvent<SVGSVGElement>) => {
      const target = e.target as SVGElement;
      const workstationEl = target.closest("[data-workstation-id]");
      // 检查是否按下了空格键（通过全局状态或事件）
      const spacePressed = (e.nativeEvent as MouseEvent & { code?: string }).code === "Space";

      if (workstationEl && !spacePressed) {
        const workstationId = workstationEl.getAttribute("data-workstation-id");
        const workstation = workstations.find((w) => w.id === workstationId);

        if (workstation && workstation.status === 0 && workstationId) {
          handleStartDrag(workstationId, e.clientX, e.clientY, workstation.x, workstation.y);
          setSelectedId(workstationId);
        }
      } else if (spacePressed || !workstationEl) {
        handlePanStart(e.clientX, e.clientY);
        if (!workstationEl) {
          setSelectedId(null);
        }
      }

      setContextMenu((prev) => ({ ...prev, visible: false }));
    },
    [workstations, handleStartDrag, handlePanStart]
  );

  /**
   * 适应视图
   */
  const fitToView = useCallback(() => {
    if (workstations.length === 0) return;

    let minX = Infinity,
      minY = Infinity,
      maxX = -Infinity,
      maxY = -Infinity;

    workstations.forEach((w) => {
      minX = Math.min(minX, w.x);
      minY = Math.min(minY, w.y);
      maxX = Math.max(maxX, w.x + w.width);
      maxY = Math.max(maxY, w.y + w.height);
    });

    const contentWidth = maxX - minX + 100;
    const contentHeight = maxY - minY + 100;
    const width = containerSize.width;
    const height = containerSize.height;

    const scaleX = (width - 100) / contentWidth;
    const scaleY = (height - 100) / contentHeight;
    const newScale = Math.min(scaleX, scaleY, 3);

    const centerX = (minX + maxX) / 2;
    const centerY = (minY + maxY) / 2;

    viewState.scale = newScale;
    viewState.offsetX = width / 2 - centerX * newScale;
    viewState.offsetY = height / 2 - centerY * newScale;

    fitToScreen();
  }, [workstations, containerSize, viewState, fitToScreen]);

  /**
   * 处理工位点击
   */
  const handleWorkstationClick = useCallback(
    (workstation: WorkstationNode, e: React.MouseEvent) => {
      e.stopPropagation();
      setSelectedId(workstation.id);
    },
    []
  );

  /**
   * 处理工位双击
   */
  const handleWorkstationDoubleClick = useCallback(
    (workstation: WorkstationNode, e: React.MouseEvent) => {
      e.stopPropagation();
      onEdit(workstation);
    },
    [onEdit]
  );

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
      workstation,
    });
  }, []);

  /**
   * 菜单项操作
   */
  const handleMenuAction = useCallback(
    async (action: string) => {
      if (!contextMenu.workstation) return;

      switch (action) {
        case "edit":
          onEdit(contextMenu.workstation);
          break;
        case "rotate-45":
          await onUpdatePosition([
            {
              id: contextMenu.workstation.id,
              positionX: contextMenu.workstation.x,
              positionY: contextMenu.workstation.y,
              rotation: ((contextMenu.workstation.rotation || 0) + 45) % 360,
            },
          ]);
          message.success("工位已顺时针旋转45°");
          break;
        case "rotate-90":
          await onUpdatePosition([
            {
              id: contextMenu.workstation.id,
              positionX: contextMenu.workstation.x,
              positionY: contextMenu.workstation.y,
              rotation: ((contextMenu.workstation.rotation || 0) + 90) % 360,
            },
          ]);
          message.success("工位已顺时针旋转90°");
          break;
        case "rotate-180":
          await onUpdatePosition([
            {
              id: contextMenu.workstation.id,
              positionX: contextMenu.workstation.x,
              positionY: contextMenu.workstation.y,
              rotation: ((contextMenu.workstation.rotation || 0) + 180) % 360,
            },
          ]);
          message.success("工位已旋转180°");
          break;
        case "rotate-reset":
          await onUpdatePosition([
            {
              id: contextMenu.workstation.id,
              positionX: contextMenu.workstation.x,
              positionY: contextMenu.workstation.y,
              rotation: 0,
            },
          ]);
          message.success("工位方向已重置");
          break;
        case "delete":
          message.warning("删除功能待实现");
          break;
        case "copy":
          message.warning("复制功能待实现");
          break;
      }

      setContextMenu((prev) => ({ ...prev, visible: false }));
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    [contextMenu.workstation, onEdit, onUpdatePosition]
  );

  /**
   * 渲染网格
   */
  const renderGrid = () => {
    if (!showGrid) return null;

    const gridSize = GRID_SIZE * viewState.scale;
    const offsetX = viewState.offsetX % gridSize;
    const offsetY = viewState.offsetY % gridSize;
    const width = containerSize.width;
    const height = containerSize.height;

    const verticalLines = [];
    const horizontalLines = [];

    for (let x = offsetX; x < width; x += gridSize) {
      verticalLines.push(
        <line key={`v-${x}`} x1={x} y1={0} x2={x} y2={height} stroke="#e8e8e8" strokeWidth="1" />
      );
    }

    for (let y = offsetY; y < height; y += gridSize) {
      horizontalLines.push(
        <line key={`h-${y}`} x1={0} y1={y} x2={width} y2={y} stroke="#e8e8e8" strokeWidth="1" />
      );
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
    return workstations.map((workstation) => {
      const colors = getWorkstationColor(workstation.status);
      const statusColor = getWorkstationStatusColor(workstation.status);
      const typeColor = getWorkstationTypeColor(workstation.type);
      const isSelected = selectedId === workstation.id;
      const isHovered = hoveredId === workstation.id;
      const isDragging = dragState.workstationId === workstation.id;

      const posX = isDragging && draggedNodePos ? draggedNodePos.x : workstation.x;
      const posY = isDragging && draggedNodePos ? draggedNodePos.y : workstation.y;
      const rotation = workstation.rotation || 0;

      const x = posX * viewState.scale + viewState.offsetX;
      const y = posY * viewState.scale + viewState.offsetY;
      const w = workstation.width * viewState.scale;
      const h = workstation.height * viewState.scale;

      const style: React.CSSProperties = {
        cursor: workstation.status === 0 ? "move" : "not-allowed",
        filter:
          isHovered && workstation.status === 0
            ? "drop-shadow(0 4px 8px rgba(0,0,0,0.15))"
            : undefined,
      };

      const hoverTransform =
        isHovered && workstation.status === 0
          ? `translate(${w / 2}, ${h / 2}) scale(1.05) translate(${-w / 2}, ${-h / 2})`
          : "";

      const rotationTransform = rotation !== 0 ? `rotate(${rotation}, ${w / 2}, ${h / 2})` : "";

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
            <rect
              width={w}
              height={h}
              fill={colors.main}
              stroke={isSelected ? "#ba3630" : colors.border}
              strokeWidth={isSelected ? 3 : 2}
              rx={6}
              ry={6}
              opacity={isDragging ? 0.8 : 1}
            />
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
            <rect
              x={w * 0.27}
              y={h * 0.12}
              width={w * 0.46}
              height={h * 0.31}
              fill={isHovered || isSelected ? statusColor.main : "#0a0a0a"}
              opacity={isHovered || isSelected ? 0.6 : 0.8}
              rx={1}
              ry={1}
              pointerEvents="none"
            />
            <rect
              x={w * 0.48}
              y={h * 0.45}
              width={w * 0.04}
              height={h * 0.08}
              fill="#333333"
              opacity={0.8}
              pointerEvents="none"
            />
            <rect
              x={w * 0.43}
              y={h * 0.52}
              width={w * 0.14}
              height={h * 0.02}
              fill="#2d2d2d"
              opacity={0.8}
              pointerEvents="none"
            />
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
            <g opacity={0.3} pointerEvents="none">
              <rect
                x={w * 0.16}
                y={h * 0.59}
                width={w * 0.08}
                height={h * 0.03}
                fill="#1a1a1a"
                rx={0.5}
              />
              <rect
                x={w * 0.25}
                y={h * 0.59}
                width={w * 0.08}
                height={h * 0.03}
                fill="#1a1a1a"
                rx={0.5}
              />
              <rect
                x={w * 0.34}
                y={h * 0.59}
                width={w * 0.08}
                height={h * 0.03}
                fill="#1a1a1a"
                rx={0.5}
              />
              <rect
                x={w * 0.16}
                y={h * 0.63}
                width={w * 0.08}
                height={h * 0.03}
                fill="#1a1a1a"
                rx={0.5}
              />
              <rect
                x={w * 0.25}
                y={h * 0.63}
                width={w * 0.08}
                height={h * 0.03}
                fill="#1a1a1a"
                rx={0.5}
              />
              <rect
                x={w * 0.34}
                y={h * 0.63}
                width={w * 0.08}
                height={h * 0.03}
                fill="#1a1a1a"
                rx={0.5}
              />
            </g>
            <ellipse
              cx={w * 0.7}
              cy={h * 0.62}
              rx={w * 0.08}
              ry={h * 0.06}
              fill="#4a4a4a"
              opacity={0.7}
              pointerEvents="none"
            />
            <g opacity={0.85} pointerEvents="none">
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
      { key: "edit", label: "编辑工位", icon: <InfoCircleOutlined /> },
      { type: "divider" },
      { key: "rotate-45", label: "顺时针旋转45°", icon: <InfoCircleOutlined /> },
      { key: "rotate-90", label: "顺时针旋转90°", icon: <InfoCircleOutlined /> },
      { key: "rotate-180", label: "旋转180°", icon: <InfoCircleOutlined /> },
      { key: "rotate-reset", label: "重置方向", icon: <InfoCircleOutlined /> },
      { type: "divider" },
      { key: "copy", label: "复制工位", icon: <InfoCircleOutlined /> },
      { key: "delete", label: "删除工位", icon: <InfoCircleOutlined />, danger: true },
    ];

    return (
      <Dropdown
        menu={{
          items: menuItems as MenuProps["items"],
          onClick: ({ key }) => handleMenuAction(key),
        }}
        open={contextMenu.visible}
        onOpenChange={(visible) => setContextMenu((prev) => ({ ...prev, visible }))}
        trigger={[]}
      >
        <div
          style={{
            position: "fixed",
            left: contextMenu.x,
            top: contextMenu.y,
            width: 1,
            height: 1,
            pointerEvents: "none",
          }}
        />
      </Dropdown>
    );
  };

  const selectedWorkstation = workstations.find((w) => w.id === selectedId);
  const width = containerSize.width;
  const height = containerSize.height;

  return (
    <Card
      ref={cardRef}
      className="floor-plan-editor"
      styles={{ body: { padding: 0 } }}
      style={{ height: "100%", overflow: "hidden" }}
    >
      {/* 工具栏 */}
      <div
        className="floor-plan-toolbar"
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          padding: "8px 12px",
          borderBottom: "1px solid #e9efeb",
        }}
      >
        <div className="toolbar-left">
          <Tooltip title="放大 (Ctrl+滚轮)">
            <Button
              type="text"
              icon={<ZoomInOutlined />}
              onClick={zoomIn}
              disabled={viewState.scale >= 3}
            />
          </Tooltip>
          <Tooltip title="缩小 (Ctrl+滚轮)">
            <Button
              type="text"
              icon={<ZoomOutOutlined />}
              onClick={zoomOut}
              disabled={viewState.scale <= 0.25}
            />
          </Tooltip>
          <span className="zoom-level">{Math.round(viewState.scale * 100)}%</span>
          <Tooltip title="适应视图">
            <Button type="text" icon={<CompressOutlined />} onClick={fitToView} />
          </Tooltip>
          <Tooltip title="重置视图">
            <Button type="text" icon={<ReloadOutlined />} onClick={resetView} />
          </Tooltip>
          <Tooltip title={showGrid ? "隐藏网格" : "显示网格"}>
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
        height={height}
        onMouseDown={handleMouseDown}
        style={{
          cursor: viewState.isDragging ? "grabbing" : "default",
          background: "#f5f5f5",
          display: "block",
        }}
      >
        <defs>
          <linearGradient id="deskHighlight" x1="0%" y1="0%" x2="0%" y2="100%">
            <stop offset="0%" stopColor="#ffffff" stopOpacity={0.4} />
            <stop offset="100%" stopColor="#ffffff" stopOpacity={0} />
          </linearGradient>
        </defs>

        <rect width={width} height={height} fill="#fafafa" />
        {renderGrid()}
        {renderWorkstations()}

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
            <text x={width / 2} y={37} textAnchor="middle" fill="#fff" fontSize={14}>
              X: {Math.round(dragState.originalX)}, Y: {Math.round(dragState.originalY)}
            </text>
          </g>
        )}
      </svg>

      {renderContextMenu()}
    </Card>
  );
};

export default FloorPlanEditor;
