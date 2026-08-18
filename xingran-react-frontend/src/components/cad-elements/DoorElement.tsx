/**
 * 门元素组件 - CAD 标准制图样式
 */

import { useMemo, useCallback } from "react";
import { getDoorColor } from "@/components/cad-editor/theme";
import type { Door } from "@/components/cad-editor/types";

export interface DoorElementProps {
  door: Door;
  selected?: boolean;
  hovered?: boolean;
  onSelect?: () => void;
  onHover?: (hovered: boolean) => void;
  onDoubleClick?: () => void;
  style?: React.CSSProperties;
}

// CAD 标准常量
const _DOOR_THICKNESS = 5;
const OPENING_STROKE_WIDTH = 12;
const OPENING_COLOR = "#ffffff";
const LINE_COLOR = "#2c3e50";
const HINGE_RADIUS = 2.5;
const ARC_STROKE_WIDTH = 2;
const FRAME_STROKE_WIDTH = 1;
const CONTROL_POINT_RADIUS = 4;
const CONTROL_POINT_STROKE_WIDTH = 2;
const SMALL_CONTROL_POINT_RADIUS = 3;
const SMALL_CONTROL_POINT_STROKE_WIDTH = 1.5;
const EMERGENCY_LABEL_OFFSET = 12;
const EMERGENCY_LABEL_COLOR = "#e74c3c";
const EMERGENCY_FONT_SIZE = 9;
const HIGHLIGHT_COLOR = "#337ab0";
const HOVER_COLOR = "#40a9ff";
const SLIDING_ARROW_LENGTH = 6;
const SLIDING_DASH_ARRAY = "4,3";

interface DoorGeometry {
  hingePoint: { x: number; y: number };
  openEndPoint: { x: number; y: number };
  leafEndX: number;
  leafEndY: number;
  leafEndX2?: number;
  leafEndY2?: number;
}

export function DoorElement({
  door,
  selected = false,
  hovered = false,
  onSelect,
  onHover,
  onDoubleClick,
  style,
}: DoorElementProps) {
  const color = useMemo(() => getDoorColor(door, selected, hovered), [door, selected, hovered]);

  const highlightColor = selected ? HIGHLIGHT_COLOR : hovered ? HOVER_COLOR : color;

  const geometry = useMemo<DoorGeometry>(() => {
    const { position, width, length, angle, direction } = door;
    const angleRad = (angle * Math.PI) / 180;
    const cos = Math.cos(angleRad);
    const sin = Math.sin(angleRad);
    const halfWidth = width / 2;

    // 计算合页和开口终点
    const hingePoint = {
      x: direction === "left" ? position.x + cos * halfWidth : position.x - cos * halfWidth,
      y: direction === "left" ? position.y + sin * halfWidth : position.y - sin * halfWidth,
    };

    const openEndPoint = {
      x: direction === "left" ? position.x - cos * halfWidth : position.x + cos * halfWidth,
      y: direction === "left" ? position.y - sin * halfWidth : position.y + sin * halfWidth,
    };

    // 计算门扇角度
    let doorAngle = angle;
    switch (direction) {
      case "left":
        doorAngle += 90;
        break;
      case "right":
        doorAngle -= 90;
        break;
    }

    const leafAngleRad = (doorAngle * Math.PI) / 180;
    const leafEndX = hingePoint.x + Math.cos(leafAngleRad) * length;
    const leafEndY = hingePoint.y + Math.sin(leafAngleRad) * length;

    const result: DoorGeometry = {
      hingePoint,
      openEndPoint,
      leafEndX,
      leafEndY,
    };

    // 双开门的第二扇
    if (door.type === "double") {
      result.leafEndX2 = openEndPoint.x + Math.cos(leafAngleRad + Math.PI) * length;
      result.leafEndY2 = openEndPoint.y + Math.sin(leafAngleRad + Math.PI) * length;
    }

    return result;
    // eslint-disable-next-line react-hooks/exhaustive-deps -- door prop accessed via nested fields
  }, [door.position, door.width, door.length, door.angle, door.direction, door.type]);

  const handleMouseEnter = useCallback(() => {
    onHover?.(true);
  }, [onHover]);

  const handleMouseLeave = useCallback(() => {
    onHover?.(false);
  }, [onHover]);

  const containerClassName = [
    "cad-door",
    selected && "cad-door-selected",
    hovered && "cad-door-hovered",
  ]
    .filter(Boolean)
    .join(" ");

  const renderDoorArc = function () {
    if (door.type === "sliding") {
      // 推拉门 - 虚线箭头
      return (
        <g style={{ pointerEvents: "none" }}>
          <line
            x1={geometry.hingePoint.x}
            y1={geometry.hingePoint.y}
            x2={geometry.leafEndX}
            y2={geometry.leafEndY}
            stroke={highlightColor}
            strokeWidth={ARC_STROKE_WIDTH}
            strokeDasharray={SLIDING_DASH_ARRAY}
          />
          <polygon
            points={`${geometry.leafEndX},${geometry.leafEndY} ${geometry.leafEndX - SLIDING_ARROW_LENGTH},${geometry.leafEndY + 3} ${geometry.leafEndX - SLIDING_ARROW_LENGTH},${geometry.leafEndY - 3}`}
            fill={highlightColor}
          />
        </g>
      );
    }

    if (
      door.type === "double" &&
      geometry.leafEndX2 !== undefined &&
      geometry.leafEndY2 !== undefined
    ) {
      // 双开门
      const sweepFlag = door.direction === "left" ? 1 : 0;
      return (
        <g style={{ pointerEvents: "none" }}>
          <path
            d={`M ${geometry.hingePoint.x} ${geometry.hingePoint.y}
                    A ${door.length} ${door.length} 0 0 ${sweepFlag}
                    ${geometry.leafEndX} ${geometry.leafEndY}`}
            fill="none"
            stroke={highlightColor}
            strokeWidth={ARC_STROKE_WIDTH}
          />
          <path
            d={`M ${geometry.openEndPoint.x} ${geometry.openEndPoint.y}
                    A ${door.length} ${door.length} 0 0 ${1 - sweepFlag}
                    ${geometry.leafEndX2} ${geometry.leafEndY2}`}
            fill="none"
            stroke={highlightColor}
            strokeWidth={ARC_STROKE_WIDTH}
          />
        </g>
      );
    }

    // 单开门
    const sweepFlag = door.direction === "left" ? 1 : 0;
    return (
      <path
        d={`M ${geometry.hingePoint.x} ${geometry.hingePoint.y}
                A ${door.length} ${door.length} 0 0 ${sweepFlag}
                ${geometry.leafEndX} ${geometry.leafEndY}`}
        fill="none"
        stroke={highlightColor}
        strokeWidth={ARC_STROKE_WIDTH}
        style={{ pointerEvents: "none" }}
      />
    );
  };

  return (
    <g
      className={containerClassName}
      onClick={onSelect}
      onDoubleClick={onDoubleClick}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      style={style}
    >
      {/* 门洞（墙体开口） */}
      <line
        x1={geometry.hingePoint.x}
        y1={geometry.hingePoint.y}
        x2={geometry.openEndPoint.x}
        y2={geometry.openEndPoint.y}
        stroke={OPENING_COLOR}
        strokeWidth={OPENING_STROKE_WIDTH}
        style={{ pointerEvents: "none" }}
      />

      {/* 门扇弧线 */}
      {renderDoorArc()}

      {/* 门框线 */}
      <line
        x1={geometry.hingePoint.x}
        y1={geometry.hingePoint.y}
        x2={geometry.openEndPoint.x}
        y2={geometry.openEndPoint.y}
        stroke={LINE_COLOR}
        strokeWidth={FRAME_STROKE_WIDTH}
        style={{ pointerEvents: "none" }}
      />

      {/* 合页标记 */}
      <circle
        cx={geometry.hingePoint.x}
        cy={geometry.hingePoint.y}
        r={HINGE_RADIUS}
        fill={LINE_COLOR}
        style={{ pointerEvents: "none" }}
      />

      {/* 控制点 */}
      {selected && (
        <>
          <circle
            cx={door.position.x}
            cy={door.position.y}
            r={CONTROL_POINT_RADIUS}
            fill="#fff"
            stroke={HIGHLIGHT_COLOR}
            strokeWidth={CONTROL_POINT_STROKE_WIDTH}
            style={{ cursor: "move" }}
          />
          <circle
            cx={geometry.hingePoint.x}
            cy={geometry.hingePoint.y}
            r={SMALL_CONTROL_POINT_RADIUS}
            fill="#fff"
            stroke={HIGHLIGHT_COLOR}
            strokeWidth={SMALL_CONTROL_POINT_STROKE_WIDTH}
            style={{ cursor: "pointer" }}
          />
          <circle
            cx={geometry.openEndPoint.x}
            cy={geometry.openEndPoint.y}
            r={SMALL_CONTROL_POINT_RADIUS}
            fill="#fff"
            stroke={HIGHLIGHT_COLOR}
            strokeWidth={SMALL_CONTROL_POINT_STROKE_WIDTH}
            style={{ cursor: "pointer" }}
          />
        </>
      )}

      {/* 紧急出口标识 */}
      {door.type === "emergency" && (
        <text
          x={door.position.x}
          y={door.position.y - door.length / 2 - EMERGENCY_LABEL_OFFSET}
          fill={EMERGENCY_LABEL_COLOR}
          fontSize={EMERGENCY_FONT_SIZE}
          textAnchor="middle"
          fontWeight="bold"
          style={{ pointerEvents: "none" }}
        >
          E
        </text>
      )}
    </g>
  );
}
