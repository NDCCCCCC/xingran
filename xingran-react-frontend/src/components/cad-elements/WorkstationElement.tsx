/**
 * 工位元素组件 - CAD 标准制图样式（基于实际办公空间设计）
 * 支持：一字型桌和L型桌，带有显示器、键盘、鼠标等设备符号
 */

import { useMemo, useCallback } from "react";
import type { WorkstationNode } from "@/components/shared/FloorPlanEditor.types";

export interface WorkstationElementProps {
  workstation: WorkstationNode;
  selected?: boolean;
  hovered?: boolean;
  onSelect?: () => void;
  onHover?: (hovered: boolean) => void;
  onDoubleClick?: () => void;
  style?: React.CSSProperties;
}

// 点类型定义
interface Point {
  x: number;
  y: number;
}

// CAD 标准颜色常量
const LINE_COLOR = "#2c3e50";
const DESK_FILL_COLOR = "rgba(236, 240, 241, 0.8)";
const DIVIDER_COLOR = "#34495e";
const DEVICE_STROKE_COLOR = "#5d6d7e";
const HIGHLIGHT_COLOR = "#337ab0";
const HOVER_COLOR = "#40a9ff";
const DEFAULT_COLOR = "#3498db";

// 椅子尺寸常量
const CHAIR_DIMENSIONS = {
  width: 32,
  depth: 32,
  backrestHeight: 6,
  armrestWidth: 5,
  armrestDepth: 10,
  deskOffset: 12,
  lShapedRotation: (135 * Math.PI) / 180,
} as const;

// 工位类型
type DeskType = "straight" | "l-shaped";

export function WorkstationElement({
  workstation,
  selected = false,
  hovered = false,
  onSelect,
  onHover,
  onDoubleClick,
  style,
}: WorkstationElementProps) {
  const highlightColor = selected ? HIGHLIGHT_COLOR : hovered ? HOVER_COLOR : DEFAULT_COLOR;

  const deskWidth = workstation.width ?? 160;
  const deskDepth = workstation.height ?? 70;
  const rotation = workstation.rotation ?? 0;
  const deskType: DeskType = workstation.type === 1 ? "l-shaped" : "straight";
  const isLShaped = deskType === "l-shaped";

  // 计算工位位置
  const geometry = useMemo(() => {
    const centerX = workstation.x || 0;
    const centerY = workstation.y || 0;

    // 旋转角度
    const angleRad = (rotation * Math.PI) / 180;
    const cos = Math.cos(angleRad);
    const sin = Math.sin(angleRad);

    // 计算旋转后的点
    function rotatePoint(x: number, y: number): Point {
      return {
        x: centerX + x * cos - y * sin,
        y: centerY + x * sin + y * cos,
      };
    }

    // 创建旋转后的矩形（辅助函数）
    function createRotatedRect(x: number, y: number, width: number, height: number): Point[] {
      const halfW = width / 2;
      const halfH = height / 2;
      return [
        rotatePoint(x - halfW, y - halfH),
        rotatePoint(x + halfW, y - halfH),
        rotatePoint(x + halfW, y + halfH),
        rotatePoint(x - halfW, y + halfH),
      ];
    }

    // 桌面矩形
    const halfDeskW = deskWidth / 2;
    const halfDeskD = deskDepth / 2;
    let deskRect: Point[];
    let lShapeRect: Point[];
    let chairCenterX = 0;
    let monitorCenterX = 0;
    let chairFrontDeskD: number;
    let monitorBackDeskD: number;
    let labelYOffset: number;

    if (isLShaped) {
      // L型桌参数
      const mainWidth = 160;
      const mainDepth = deskDepth;
      const sideWidth = 60;
      const sideDepth = 80;
      const mainLeft = -mainWidth / 2;
      const mainTop = -mainDepth / 2;
      const mainRight = mainLeft + mainWidth;
      const mainBottom = mainTop + mainDepth;
      const sideLeft = mainLeft - sideWidth + 60;
      const sideTop = mainBottom;
      const sideRight = sideLeft + sideWidth;
      const sideBottom = sideTop + sideDepth;

      // 主桌面和侧边桌面
      deskRect = [
        rotatePoint(mainLeft, mainTop),
        rotatePoint(mainRight, mainTop),
        rotatePoint(mainRight, mainBottom),
        rotatePoint(mainLeft, mainBottom),
      ];
      lShapeRect = [
        rotatePoint(sideLeft, sideTop),
        rotatePoint(sideRight, sideTop),
        rotatePoint(sideRight, sideBottom),
        rotatePoint(sideLeft, sideBottom),
      ];

      // 椅子和显示器位置
      chairCenterX = sideRight + 40;
      monitorCenterX = (mainLeft + mainRight) / 2;
      chairFrontDeskD = mainBottom + 30;
      monitorBackDeskD = mainDepth;
      labelYOffset = sideBottom + 15;
    } else {
      // 一字型桌
      deskRect = [
        rotatePoint(-halfDeskW, -halfDeskD),
        rotatePoint(halfDeskW, -halfDeskD),
        rotatePoint(halfDeskW, halfDeskD),
        rotatePoint(-halfDeskW, halfDeskD),
      ];
      lShapeRect = [];
      chairCenterX = 0;
      monitorCenterX = 0;
      chairFrontDeskD = halfDeskD;
      monitorBackDeskD = halfDeskD;
      labelYOffset = 0; // 将在下面计算
    }

    // 椅子中心Y坐标
    const chairCenterY = chairFrontDeskD + CHAIR_DIMENSIONS.deskOffset;
    if (!isLShaped) {
      labelYOffset = chairCenterY + CHAIR_DIMENSIONS.depth / 2 + 15;
    }

    // L型桌椅子的额外旋转
    const chairRotation = isLShaped ? CHAIR_DIMENSIONS.lShapedRotation : 0;
    const chairCos = Math.cos(chairRotation);
    const chairSin = Math.sin(chairRotation);

    // 椅子局部坐标旋转辅助函数
    function rotateChairLocal(x: number, y: number): Point {
      const dx = x - chairCenterX;
      const dy = y - chairCenterY;
      return {
        x: chairCenterX + dx * chairCos - dy * chairSin,
        y: chairCenterY + dx * chairSin + dy * chairCos,
      };
    }

    // 将局部矩形转换为世界坐标的通用函数
    function toWorldRect(localRect: Point[]): Point[] {
      let rect = localRect;
      if (isLShaped) {
        rect = localRect.map((p) => rotateChairLocal(p.x, p.y));
      }
      return rect.map((p) => rotatePoint(p.x, p.y));
    }

    // 椅子矩形
    const {
      width: chairW,
      depth: chairD,
      backrestHeight,
      armrestWidth,
      armrestDepth,
    } = CHAIR_DIMENSIONS;
    const chairRectLocal: Point[] = [
      { x: chairCenterX - chairW / 2, y: chairCenterY - chairD / 2 },
      { x: chairCenterX + chairW / 2, y: chairCenterY - chairD / 2 },
      { x: chairCenterX + chairW / 2, y: chairCenterY + chairD / 2 },
      { x: chairCenterX - chairW / 2, y: chairCenterY + chairD / 2 },
    ];
    const chairRect = toWorldRect(chairRectLocal);

    // 椅子靠背
    const backrestRectLocal: Point[] = [
      { x: chairCenterX - chairW / 2, y: chairCenterY - chairD / 2 - backrestHeight },
      { x: chairCenterX + chairW / 2, y: chairCenterY - chairD / 2 - backrestHeight },
      { x: chairCenterX + chairW / 2, y: chairCenterY - chairD / 2 },
      { x: chairCenterX - chairW / 2, y: chairCenterY - chairD / 2 },
    ];
    const backrestRect = toWorldRect(backrestRectLocal);

    // 左扶手
    const leftArmrestLocal: Point[] = [
      { x: chairCenterX - chairW / 2 - armrestWidth / 2, y: chairCenterY - armrestDepth / 2 },
      { x: chairCenterX - chairW / 2 + armrestWidth / 2, y: chairCenterY - armrestDepth / 2 },
      { x: chairCenterX - chairW / 2 + armrestWidth / 2, y: chairCenterY + armrestDepth / 2 },
      { x: chairCenterX - chairW / 2 - armrestWidth / 2, y: chairCenterY + armrestDepth / 2 },
    ];
    const leftArmrest = toWorldRect(leftArmrestLocal);

    // 右扶手
    const rightArmrestLocal: Point[] = [
      { x: chairCenterX + chairW / 2 - armrestWidth / 2, y: chairCenterY - armrestDepth / 2 },
      { x: chairCenterX + chairW / 2 + armrestWidth / 2, y: chairCenterY - armrestDepth / 2 },
      { x: chairCenterX + chairW / 2 + armrestWidth / 2, y: chairCenterY + armrestDepth / 2 },
      { x: chairCenterX + chairW / 2 - armrestWidth / 2, y: chairCenterY + armrestDepth / 2 },
    ];
    const rightArmrest = toWorldRect(rightArmrestLocal);

    // 显示器、键盘、鼠标（使用辅助函数简化）
    const monitorOffset = monitorBackDeskD - 14;
    const monitorY = -monitorBackDeskD + monitorOffset;
    const monitorRect = createRotatedRect(monitorCenterX, monitorY, 36, 3);

    // 显示器支架
    const standRect = createRotatedRect(monitorCenterX, monitorY + 3 / 2 + 3, 6, 6);

    // 键盘
    const keyboardOffset = monitorOffset + 5;
    const keyboardY = -monitorBackDeskD + keyboardOffset;
    const keyboardRect = createRotatedRect(monitorCenterX, keyboardY, 32, 11);

    // 鼠标
    const mouseOffset = keyboardOffset + 8;
    const mouseY = -monitorBackDeskD + mouseOffset;
    const mouseRect = createRotatedRect(monitorCenterX, mouseY, 5, 5);

    // 隔断
    const partitionWidth = 3;
    let leftPartition: Point[];
    let rightPartition: Point[];

    if (isLShaped) {
      leftPartition = [];
      rightPartition = [];
    } else {
      leftPartition = createRotatedRect(
        -halfDeskW - partitionWidth / 2,
        0,
        partitionWidth,
        deskDepth
      );
      rightPartition = createRotatedRect(
        halfDeskW + partitionWidth / 2,
        0,
        partitionWidth,
        deskDepth
      );
    }

    // 标签和状态指示器位置
    const labelPos = rotatePoint(0, labelYOffset);
    const statusPos = rotatePoint(0, -deskDepth / 2 - 15);

    return {
      deskRect,
      lShapeRect,
      chairRect,
      backrestRect,
      leftArmrest,
      rightArmrest,
      monitorRect,
      standRect,
      keyboardRect,
      mouseRect,
      leftPartition,
      rightPartition,
      centerX,
      centerY,
      chairCenterX,
      chairCenterY,
      labelCenterX: labelPos.x,
      labelCenterY: labelPos.y,
      statusIndicatorX: statusPos.x,
      statusIndicatorY: statusPos.y,
    };
  }, [workstation.x, workstation.y, deskWidth, deskDepth, isLShaped, rotation]);

  const handleMouseEnter = useCallback(() => {
    onHover?.(true);
  }, [onHover]);

  const handleMouseLeave = useCallback(() => {
    onHover?.(false);
  }, [onHover]);

  // 生成多边形路径字符串的辅助函数
  const toPointsString = function (points: Point[]): string {
    return points.map((p) => `${p.x},${p.y}`).join(" ");
  };

  // 生成所有路径字符串
  const deskPoints = toPointsString(geometry.deskRect);
  const lShapePoints = toPointsString(geometry.lShapeRect);
  const chairPoints = toPointsString(geometry.chairRect);
  const backrestPoints = toPointsString(geometry.backrestRect);
  const leftArmrestPoints = toPointsString(geometry.leftArmrest);
  const rightArmrestPoints = toPointsString(geometry.rightArmrest);
  const _monitorPoints = toPointsString(geometry.monitorRect);
  const standPoints = toPointsString(geometry.standRect);
  const keyboardPoints = toPointsString(geometry.keyboardRect);
  const mousePoints = toPointsString(geometry.mouseRect);
  const leftPartitionPoints = toPointsString(geometry.leftPartition);
  const rightPartitionPoints = toPointsString(geometry.rightPartition);

  const containerClassName = [
    "cad-workstation",
    selected && "cad-workstation-selected",
    hovered && "cad-workstation-hovered",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <g
      className={containerClassName}
      onClick={onSelect}
      onDoubleClick={onDoubleClick}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      style={style}
    >
      {/* L型桌面 */}
      {isLShaped && (
        <polygon
          points={lShapePoints}
          fill={DESK_FILL_COLOR}
          stroke={highlightColor}
          strokeWidth={selected ? 2 : 1.2}
          style={{ pointerEvents: "none" }}
        />
      )}

      {/* 主桌面 */}
      <polygon
        points={deskPoints}
        fill={DESK_FILL_COLOR}
        stroke={highlightColor}
        strokeWidth={selected ? 2 : 1.2}
        style={{ cursor: "pointer" }}
      />

      {/* 桌面内边框装饰线 */}
      <polygon
        points={deskPoints}
        fill="none"
        stroke={LINE_COLOR}
        strokeWidth={0.4}
        strokeDasharray="2,2"
        opacity={0.25}
        style={{ pointerEvents: "none" }}
      />

      {/* 隔断（左右两侧） */}
      {leftPartitionPoints && (
        <polygon
          points={leftPartitionPoints}
          fill={DIVIDER_COLOR}
          stroke="none"
          style={{ pointerEvents: "none" }}
        />
      )}
      {rightPartitionPoints && (
        <polygon
          points={rightPartitionPoints}
          fill={DIVIDER_COLOR}
          stroke="none"
          style={{ pointerEvents: "none" }}
        />
      )}

      {/* 显示器支架 */}
      <polygon
        points={standPoints}
        fill={DEVICE_STROKE_COLOR}
        stroke="none"
        style={{ pointerEvents: "none" }}
      />

      {/* 显示器 */}
      <rect
        x={geometry.monitorRect[0].x}
        y={geometry.monitorRect[0].y}
        width={Math.abs(geometry.monitorRect[1].x - geometry.monitorRect[0].x)}
        height={Math.abs(geometry.monitorRect[1].y - geometry.monitorRect[0].y)}
        fill={highlightColor}
        stroke="none"
        rx={1}
        style={{ pointerEvents: "none" }}
      />

      {/* 键盘 */}
      <polygon
        points={keyboardPoints}
        fill="none"
        stroke={DEVICE_STROKE_COLOR}
        strokeWidth={0.6}
        style={{ pointerEvents: "none" }}
      />

      {/* 鼠标 */}
      <polygon
        points={mousePoints}
        fill="none"
        stroke={DEVICE_STROKE_COLOR}
        strokeWidth={0.5}
        style={{ pointerEvents: "none" }}
      />

      {/* 椅子 */}
      <g style={{ pointerEvents: "none" }}>
        <polygon
          points={backrestPoints}
          fill={DEVICE_STROKE_COLOR}
          stroke={LINE_COLOR}
          strokeWidth={0.8}
        />
        <polygon
          points={chairPoints}
          fill="rgba(52, 152, 219, 0.2)"
          stroke={DEVICE_STROKE_COLOR}
          strokeWidth={0.8}
        />
        <polygon
          points={leftArmrestPoints}
          fill={DEVICE_STROKE_COLOR}
          stroke={LINE_COLOR}
          strokeWidth={0.5}
        />
        <polygon
          points={rightArmrestPoints}
          fill={DEVICE_STROKE_COLOR}
          stroke={LINE_COLOR}
          strokeWidth={0.5}
        />
      </g>

      {/* 工位编号/名称 */}
      {(workstation.name || workstation.id) && (
        <text
          x={geometry.labelCenterX}
          y={geometry.labelCenterY}
          fill={highlightColor}
          fontSize={12}
          fontWeight="500"
          textAnchor="middle"
          dominantBaseline="hanging"
          style={{ pointerEvents: "none" }}
        >
          {workstation.name || workstation.id.slice(0, 6)}
        </text>
      )}

      {/* 状态指示器 */}
      <g style={{ pointerEvents: "none" }}>
        <circle
          cx={geometry.statusIndicatorX}
          cy={geometry.statusIndicatorY}
          r={7}
          fill={highlightColor}
          stroke={LINE_COLOR}
          strokeWidth={1}
        />
        <text
          x={geometry.statusIndicatorX}
          y={geometry.statusIndicatorY}
          fill="#fff"
          fontSize={9}
          fontWeight="bold"
          textAnchor="middle"
          dominantBaseline="middle"
        >
          {workstation.status === 0 ? "空" : workstation.status === 1 ? "占" : "维"}
        </text>
      </g>

      {/* 选中状态的控制点 */}
      {selected && (
        <>
          {geometry.deskRect.map((point, index) => (
            <circle
              key={`corner-${index}`}
              cx={point.x}
              cy={point.y}
              r={3.5}
              fill="#fff"
              stroke={HIGHLIGHT_COLOR}
              strokeWidth={1.5}
              className="cad-workstation-control-point"
              style={{ cursor: "move" }}
            />
          ))}
          <circle
            cx={geometry.centerX}
            cy={geometry.centerY}
            r={4}
            fill="#fff"
            stroke={HIGHLIGHT_COLOR}
            strokeWidth={2}
            className="cad-workstation-center-point"
            style={{ cursor: "grab" }}
          />
        </>
      )}

      {/* 方向指示器 */}
      {rotation !== 0 && (
        <g style={{ pointerEvents: "none" }}>
          <line
            x1={geometry.labelCenterX}
            y1={geometry.labelCenterY - 5}
            x2={geometry.labelCenterX}
            y2={geometry.labelCenterY - 20}
            stroke={LINE_COLOR}
            strokeWidth={1}
          />
          <polygon
            points={`${geometry.labelCenterX},${geometry.labelCenterY - 23} ${geometry.labelCenterX - 3},${geometry.labelCenterY - 16} ${geometry.labelCenterX + 3},${geometry.labelCenterY - 16}`}
            fill={LINE_COLOR}
          />
        </g>
      )}
    </g>
  );
}
