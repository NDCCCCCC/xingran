/**
 * 工位元素组件 - CAD 标准制图样式（基于实际办公空间设计）
 * 支持：一字型桌和L型桌，带有显示器、键盘、鼠标等设备符号
 */

import { useMemo } from 'react';
import { getWorkstationColor } from '@/components/cad-editor/theme';
import type { WorkstationNode } from '@/components/shared/FloorPlanEditor.types';

export interface WorkstationElementProps {
  workstation: WorkstationNode;
  selected?: boolean;
  hovered?: boolean;
  onSelect?: () => void;
  onHover?: (hovered: boolean) => void;
  onDoubleClick?: () => void;
  style?: React.CSSProperties;
}

// 工位类型定义
type DeskType = 'straight' | 'l-shaped';

export const WorkstationElement: React.FC<WorkstationElementProps> = ({
  workstation,
  selected = false,
  hovered = false,
  onSelect,
  onHover,
  onDoubleClick,
  style,
}) => {
  // CAD 标准颜色
  const lineColor = '#2c3e50'; // 深灰色线条
  const deskFillColor = 'rgba(236, 240, 241, 0.8)'; // 桌面填充色
  const dividerColor = '#34495e'; // 隔断颜色
  const deviceStrokeColor = '#5d6d7e'; // 设备边框色
  const highlightColor = selected ? '#1890ff' : hovered ? '#40a9ff' : '#3498db';

  // 工位尺寸（CAD 标准：毫米）
  const deskWidth = workstation.width || 160; // 默认1600mm（加宽）
  const deskDepth = workstation.height || 70; // 默认700mm
  const rotation = workstation.rotation || 0;
  const deskType: DeskType = (workstation.type === 1 ? 'l-shaped' : 'straight');

  // 计算工位位置
  const geometry = useMemo(() => {
    const centerX = workstation.x || 0;
    const centerY = workstation.y || 0;

    // 旋转角度
    const angleRad = (rotation * Math.PI) / 180;
    const cos = Math.cos(angleRad);
    const sin = Math.sin(angleRad);

    // 计算旋转后的点
    function rotatePoint(x: number, y: number) {
      const dx = x;
      const dy = y;
      return {
        x: centerX + dx * cos - dy * sin,
        y: centerY + dx * sin + dy * cos,
      };
    }

    // L型桌参数（基于deskDepth计算）
    const mainWidth = 160; // 主桌面宽度
    const mainDepth = deskDepth; // 主桌面深度，使用deskDepth
    const sideWidth = 60; // 侧边桌面宽度
    const sideDepth = 80; // 侧边桌面深度

    // L型桌坐标（定义在条件块外以便后续使用）
    // 主桌面：居中放置
    const mainLeft = -mainWidth / 2; // -80
    const mainTop = -mainDepth / 2; // -40
    const mainRight = mainLeft + mainWidth; // 80
    const mainBottom = mainTop + mainDepth; // 40

    // 侧边桌面：连接在主桌面左侧下方，向右移动60
    const sideLeft = mainLeft - sideWidth + 60; // -80
    const sideTop = mainBottom; // 40（紧贴主桌面下边）
    const sideRight = sideLeft + sideWidth; // -20
    const sideBottom = sideTop + sideDepth; // 100

    // 椅子和显示器中心位置（在条件块之前定义以便在return中使用）
    let chairCenterX = 0;
    let monitorCenterX = 0;

    // 一字型桌：完整矩形
    const halfDeskW = deskWidth / 2;
    const halfDeskD = deskDepth / 2;
    // 定义点类型
    interface Point {
      x: number;
      y: number;
    }
    let deskRect: Point[] = [];
    let lShapeRect: Point[] = [];

    if (deskType === 'straight') {
      // 一字型桌：单个矩形
      deskRect = [
        rotatePoint(-halfDeskW, -halfDeskD),
        rotatePoint(halfDeskW, -halfDeskD),
        rotatePoint(halfDeskW, halfDeskD),
        rotatePoint(-halfDeskW, halfDeskD),
      ];
      lShapeRect = [];
      chairCenterX = 0;
      monitorCenterX = 0;
    } else {
      // L型桌：主桌面和侧边桌面连接成L形
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

      // L型桌：椅子在L型转角处，向右下方移动避免重叠
      chairCenterX = sideRight + 40; // 在侧边桌右边再向右40
      monitorCenterX = (mainLeft + mainRight) / 2; // 显示器在主桌面中心
    }

    // 椅子位置计算
    let chairCenterY = 0;
    let chairFrontDeskD = halfDeskD; // 椅子前方的桌边深度

    if (deskType === 'straight') {
      // 一字型桌：椅子位于桌面中心正下方
      chairFrontDeskD = halfDeskD;
    } else {
      // L型桌：椅子在转角处，向下移动
      chairFrontDeskD = mainBottom + 30; // 在主桌面下边再下移30
    }

    const chairOffset = 12; // 椅子距离桌边的距离
    const chairWidth = 32; // 椅子宽度
    const chairDepth = 32; // 椅子深度
    chairCenterY = chairFrontDeskD + chairOffset;

    // L型桌椅子的额外旋转角度（90度，顺时针）
    const chairRotation = deskType === 'l-shaped' ? 135 * Math.PI / 180 : 0;
    const chairCos = Math.cos(chairRotation);
    const chairSin = Math.sin(chairRotation);

    // 为L型桌椅子应用旋转的辅助函数
    function rotateChairPoint(x: number, y: number) {
      const dx = x - chairCenterX;
      const dy = y - chairCenterY;
      return {
        x: chairCenterX + dx * chairCos - dy * chairSin,
        y: chairCenterY + dx * chairSin + dy * chairCos,
      };
    }

    // 椅子矩形（俯视）- 使用世界坐标系
    // 对于 L 型桌，椅子需要旋转，所以先创建局部坐标的椅子，然后旋转，最后转换为世界坐标
    const _chairRectLocal = [
      { x: chairCenterX - chairWidth/2, y: chairCenterY - chairDepth/2 },
      { x: chairCenterX + chairWidth/2, y: chairCenterY - chairDepth/2 },
      { x: chairCenterX + chairWidth/2, y: chairCenterY + chairDepth/2 },
      { x: chairCenterX - chairWidth/2, y: chairCenterY + chairDepth/2 },
    ];

    let chairRect: { x: number; y: number }[];
    if (deskType === 'l-shaped') {
      // L 型桌：先旋转，再转换为世界坐标
      const rotatedChairRect = [
        rotateChairPoint(_chairRectLocal[0].x, _chairRectLocal[0].y),
        rotateChairPoint(_chairRectLocal[1].x, _chairRectLocal[1].y),
        rotateChairPoint(_chairRectLocal[2].x, _chairRectLocal[2].y),
        rotateChairPoint(_chairRectLocal[3].x, _chairRectLocal[3].y),
      ];
      // 旋转后的坐标仍然是局部坐标，需要转换为世界坐标
      chairRect = rotatedChairRect.map(p => rotatePoint(p.x, p.y));
    } else {
      // 一字型桌：直接转换为世界坐标
      chairRect = _chairRectLocal.map(p => rotatePoint(p.x, p.y));
    }

    // 椅子靠背（上方）
    const backrestHeight = 6;
    const _backrestRectLocal = [
      { x: chairCenterX - chairWidth/2, y: chairCenterY - chairDepth/2 - backrestHeight },
      { x: chairCenterX + chairWidth/2, y: chairCenterY - chairDepth/2 - backrestHeight },
      { x: chairCenterX + chairWidth/2, y: chairCenterY - chairDepth/2 },
      { x: chairCenterX - chairWidth/2, y: chairCenterY - chairDepth/2 },
    ];

    let backrestRect: { x: number; y: number }[];
    if (deskType === 'l-shaped') {
      const rotatedBackrestRect = [
        rotateChairPoint(_backrestRectLocal[0].x, _backrestRectLocal[0].y),
        rotateChairPoint(_backrestRectLocal[1].x, _backrestRectLocal[1].y),
        rotateChairPoint(_backrestRectLocal[2].x, _backrestRectLocal[2].y),
        rotateChairPoint(_backrestRectLocal[3].x, _backrestRectLocal[3].y),
      ];
      backrestRect = rotatedBackrestRect.map(p => rotatePoint(p.x, p.y));
    } else {
      backrestRect = _backrestRectLocal.map(p => rotatePoint(p.x, p.y));
    }

    // 扶手（左右两侧）
    const armrestWidth = 5;
    const armrestDepth = 10;
    const _leftArmrestLocal = [
      { x: chairCenterX - chairWidth/2 - armrestWidth/2, y: chairCenterY - armrestDepth/2 },
      { x: chairCenterX - chairWidth/2 + armrestWidth/2, y: chairCenterY - armrestDepth/2 },
      { x: chairCenterX - chairWidth/2 + armrestWidth/2, y: chairCenterY + armrestDepth/2 },
      { x: chairCenterX - chairWidth/2 - armrestWidth/2, y: chairCenterY + armrestDepth/2 },
    ];

    let leftArmrest: { x: number; y: number }[];
    if (deskType === 'l-shaped') {
      const rotatedLeftArmrest = [
        rotateChairPoint(_leftArmrestLocal[0].x, _leftArmrestLocal[0].y),
        rotateChairPoint(_leftArmrestLocal[1].x, _leftArmrestLocal[1].y),
        rotateChairPoint(_leftArmrestLocal[2].x, _leftArmrestLocal[2].y),
        rotateChairPoint(_leftArmrestLocal[3].x, _leftArmrestLocal[3].y),
      ];
      leftArmrest = rotatedLeftArmrest.map(p => rotatePoint(p.x, p.y));
    } else {
      leftArmrest = _leftArmrestLocal.map(p => rotatePoint(p.x, p.y));
    }

    const _rightArmrestLocal = [
      { x: chairCenterX + chairWidth/2 - armrestWidth/2, y: chairCenterY - armrestDepth/2 },
      { x: chairCenterX + chairWidth/2 + armrestWidth/2, y: chairCenterY - armrestDepth/2 },
      { x: chairCenterX + chairWidth/2 + armrestWidth/2, y: chairCenterY + armrestDepth/2 },
      { x: chairCenterX + chairWidth/2 - armrestWidth/2, y: chairCenterY + armrestDepth/2 },
    ];

    let rightArmrest: { x: number; y: number }[];
    if (deskType === 'l-shaped') {
      const rotatedRightArmrest = [
        rotateChairPoint(_rightArmrestLocal[0].x, _rightArmrestLocal[0].y),
        rotateChairPoint(_rightArmrestLocal[1].x, _rightArmrestLocal[1].y),
        rotateChairPoint(_rightArmrestLocal[2].x, _rightArmrestLocal[2].y),
        rotateChairPoint(_rightArmrestLocal[3].x, _rightArmrestLocal[3].y),
      ];
      rightArmrest = rotatedRightArmrest.map(p => rotatePoint(p.x, p.y));
    } else {
      rightArmrest = _rightArmrestLocal.map(p => rotatePoint(p.x, p.y));
    }

    // 显示器、键盘、鼠标位置计算
    let monitorBackDeskD = halfDeskD; // 显示器后方的桌边深度
    if (deskType === 'straight') {
      monitorBackDeskD = halfDeskD;
    } else {
      // L型桌：显示器在主桌面后端
      monitorBackDeskD = mainDepth; // 主桌面的深度
    }

    // 显示器（位于主桌面后端中央）
    const monitorOffset = monitorBackDeskD - 14; // 距离桌边
    const monitorWidth = 36;
    const monitorHeight = 3; // 显示器厚度（俯视图为线）
    const monitorY = -monitorBackDeskD + monitorOffset;
    const monitorRect = [
      rotatePoint(monitorCenterX - monitorWidth/2, monitorY - monitorHeight/2),
      rotatePoint(monitorCenterX + monitorWidth/2, monitorY - monitorHeight/2),
      rotatePoint(monitorCenterX + monitorWidth/2, monitorY + monitorHeight/2),
      rotatePoint(monitorCenterX - monitorWidth/2, monitorY + monitorHeight/2),
    ];

    // 显示器支架（小矩形）
    const standWidth = 6;
    const standDepth = 6;
    const standRect = [
      rotatePoint(monitorCenterX - standWidth/2, monitorY + monitorHeight/2),
      rotatePoint(monitorCenterX + standWidth/2, monitorY + monitorHeight/2),
      rotatePoint(monitorCenterX + standWidth/2, monitorY + monitorHeight/2 + standDepth),
      rotatePoint(monitorCenterX - standWidth/2, monitorY + monitorHeight/2 + standDepth),
    ];

    // 键盘（显示器前方）
    const keyboardOffset = monitorOffset + 5;
    const keyboardWidth = 32;
    const keyboardDepth = 11;
    const keyboardY = -monitorBackDeskD + keyboardOffset;
    const keyboardRect = [
      rotatePoint(monitorCenterX - keyboardWidth/2, keyboardY - keyboardDepth/2),
      rotatePoint(monitorCenterX + keyboardWidth/2, keyboardY - keyboardDepth/2),
      rotatePoint(monitorCenterX + keyboardWidth/2, keyboardY + keyboardDepth/2),
      rotatePoint(monitorCenterX - keyboardWidth/2, keyboardY + keyboardDepth/2),
    ];

    // 鼠标（键盘右侧）
    const mouseOffset = keyboardOffset + 8;
    const mouseSize = 5;
    const mouseY = -monitorBackDeskD + mouseOffset;
    const mouseRect = [
      rotatePoint(monitorCenterX - mouseSize/2, mouseY - mouseSize/2),
      rotatePoint(monitorCenterX + mouseSize/2, mouseY - mouseSize/2),
      rotatePoint(monitorCenterX + mouseSize/2, mouseY + mouseSize/2),
      rotatePoint(monitorCenterX - mouseSize/2, mouseY + mouseSize/2),
    ];

    // 隔断（左右两侧）
    const partitionWidth = 3;
    let leftPartition: typeof deskRect = [];
    let rightPartition: typeof deskRect = [];

    if (deskType === 'straight') {
      // 一字型桌：左右隔断
      const partitionHeight = deskDepth;
      leftPartition = [
        rotatePoint(-halfDeskW - partitionWidth/2, -partitionHeight/2),
        rotatePoint(-halfDeskW + partitionWidth/2, -partitionHeight/2),
        rotatePoint(-halfDeskW + partitionWidth/2, partitionHeight/2),
        rotatePoint(-halfDeskW - partitionWidth/2, partitionHeight/2),
      ];
      rightPartition = [
        rotatePoint(halfDeskW - partitionWidth/2, -partitionHeight/2),
        rotatePoint(halfDeskW + partitionWidth/2, -partitionHeight/2),
        rotatePoint(halfDeskW + partitionWidth/2, partitionHeight/2),
        rotatePoint(halfDeskW - partitionWidth/2, partitionHeight/2),
      ];
    } else {
      // L型桌：不需要隔断
      leftPartition = [];
      rightPartition = [];
    }

    // 计算标签和状态指示器的位置（使用rotatePoint转换为绝对坐标并应用旋转）
    // 工位名称：放在工位下方（一字型：椅子下方，L型：侧边桌下方）
    // 状态指示器：保持在工位上方
    let labelYOffset = 0;
    if (deskType === 'l-shaped') {
      labelYOffset = sideBottom + 15; // L型：侧边桌下方15px
    } else {
      labelYOffset = chairCenterY + chairDepth/2 + 15; // 一字型：椅子下方15px
    }

    const labelPos = rotatePoint(0, labelYOffset); // 标签位置（绝对坐标，已应用旋转）
    const statusPos = rotatePoint(0, -deskDepth/2 - 15); // 状态指示器位置（绝对坐标，已应用旋转）

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
  }, [workstation.x, workstation.y, workstation.width, workstation.height, workstation.rotation, workstation.type, deskWidth, deskDepth, deskType]);

  const handleMouseEnter = () => {
    onHover?.(true);
  };

  const handleMouseLeave = () => {
    onHover?.(false);
  };

  // 生成多边形路径字符串
  const deskPoints = geometry.deskRect.map((p) => `${p.x},${p.y}`).join(' ');
  const lShapePoints = geometry.lShapeRect.map((p) => `${p.x},${p.y}`).join(' ');
  const chairPoints = geometry.chairRect.map((p) => `${p.x},${p.y}`).join(' ');
  const backrestPoints = geometry.backrestRect.map((p) => `${p.x},${p.y}`).join(' ');
  const leftArmrestPoints = geometry.leftArmrest.map((p) => `${p.x},${p.y}`).join(' ');
  const rightArmrestPoints = geometry.rightArmrest.map((p) => `${p.x},${p.y}`).join(' ');
  const monitorPoints = geometry.monitorRect.map((p) => `${p.x},${p.y}`).join(' ');
  const standPoints = geometry.standRect.map((p) => `${p.x},${p.y}`).join(' ');
  const keyboardPoints = geometry.keyboardRect.map((p) => `${p.x},${p.y}`).join(' ');
  const mousePoints = geometry.mouseRect.map((p) => `${p.x},${p.y}`).join(' ');
  const leftPartitionPoints = geometry.leftPartition.map((p) => `${p.x},${p.y}`).join(' ');
  const rightPartitionPoints = geometry.rightPartition.map((p) => `${p.x},${p.y}`).join(' ');

  return (
    <g
      className={`cad-workstation ${selected ? 'cad-workstation-selected' : ''} ${hovered ? 'cad-workstation-hovered' : ''}`}
      onClick={onSelect}
      onDoubleClick={onDoubleClick}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      style={style}
    >
      {/* L型桌面 */}
      {deskType === 'l-shaped' && (
        <polygon
          points={lShapePoints}
          fill={deskFillColor}
          stroke={selected ? highlightColor : lineColor}
          strokeWidth={selected ? 2 : 1.2}
          style={{ pointerEvents: 'none' }}
        />
      )}

      {/* 主桌面 */}
      <polygon
        points={deskPoints}
        fill={deskFillColor}
        stroke={selected ? highlightColor : lineColor}
        strokeWidth={selected ? 2 : 1.2}
        style={{ cursor: 'pointer' }}
      />

      {/* 桌面内边框装饰线 */}
      <polygon
        points={deskPoints}
        fill="none"
        stroke={lineColor}
        strokeWidth={0.4}
        strokeDasharray="2,2"
        opacity={0.25}
        style={{ pointerEvents: 'none' }}
      />

      {/* 隔断（左右两侧） */}
      {/* 隔断（左右两侧） */}
      {leftPartitionPoints && (
        <polygon
          points={leftPartitionPoints}
          fill={dividerColor}
          stroke="none"
          style={{ pointerEvents: 'none' }}
        />
      )}
      {rightPartitionPoints && (
        <polygon
          points={rightPartitionPoints}
          fill={dividerColor}
          stroke="none"
          style={{ pointerEvents: 'none' }}
        />
      )}

      {/* 显示器支架 */}
      <polygon
        points={standPoints}
        fill={deviceStrokeColor}
        stroke="none"
        style={{ pointerEvents: 'none' }}
      />

      {/* 显示器 */}
      <rect
        x={geometry.monitorRect[0].x}
        y={geometry.monitorRect[0].y}
        width={Math.abs(geometry.monitorRect[1].x - geometry.monitorRect[0].x)}
        height={Math.abs(geometry.monitorRect[1].y - geometry.monitorRect[0].y)}
        fill={selected ? highlightColor : deviceStrokeColor}
        stroke="none"
        rx={1}
        style={{ pointerEvents: 'none' }}
      />

      {/* 键盘 */}
      <polygon
        points={keyboardPoints}
        fill="none"
        stroke={deviceStrokeColor}
        strokeWidth={0.6}
        style={{ pointerEvents: 'none' }}
      />

      {/* 鼠标 */}
      <polygon
        points={mousePoints}
        fill="none"
        stroke={deviceStrokeColor}
        strokeWidth={0.5}
        style={{ pointerEvents: 'none' }}
      />

      {/* 椅子 - 位于桌面前方（CAD图的下方） */}
      <g style={{ pointerEvents: 'none' }}>
        {/* 椅子靠背 */}
        <polygon
          points={backrestPoints}
          fill={deviceStrokeColor}
          stroke={lineColor}
          strokeWidth={0.8}
        />
        {/* 椅子座位 */}
        <polygon
          points={chairPoints}
          fill="rgba(52, 152, 219, 0.2)"
          stroke={deviceStrokeColor}
          strokeWidth={0.8}
        />
        {/* 扶手 */}
        <polygon
          points={leftArmrestPoints}
          fill={deviceStrokeColor}
          stroke={lineColor}
          strokeWidth={0.5}
        />
        <polygon
          points={rightArmrestPoints}
          fill={deviceStrokeColor}
          stroke={lineColor}
          strokeWidth={0.5}
        />
      </g>

      {/* 工位编号/名称 */}
      {(workstation.name || workstation.id) && (
        <text
          x={geometry.labelCenterX}
          y={geometry.labelCenterY}
          fill={selected ? highlightColor : lineColor}
          fontSize={12}
          fontWeight="500"
          textAnchor="middle"
          dominantBaseline="hanging"
          style={{ pointerEvents: 'none' }}
        >
          {workstation.name || workstation.id.slice(0, 6)}
        </text>
      )}

      {/* 状态指示器 - CAD 标准符号 */}
      <g style={{ pointerEvents: 'none' }}>
        {/* 状态圆圈 */}
        <circle
          cx={geometry.statusIndicatorX}
          cy={geometry.statusIndicatorY}
          r={7}
          fill={selected ? highlightColor : getWorkstationColor(workstation, selected, hovered)}
          stroke={lineColor}
          strokeWidth={1}
        />
        {/* 状态文字 */}
        <text
          x={geometry.statusIndicatorX}
          y={geometry.statusIndicatorY}
          fill="#fff"
          fontSize={9}
          fontWeight="bold"
          textAnchor="middle"
          dominantBaseline="middle"
        >
          {workstation.status === 0 ? '空' : workstation.status === 1 ? '占' : '维'}
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
              stroke="#1890ff"
              strokeWidth={1.5}
              className="cad-workstation-control-point"
              style={{ cursor: 'move' }}
            />
          ))}
          {/* 中心控制点 */}
          <circle
            cx={geometry.centerX}
            cy={geometry.centerY}
            r={4}
            fill="#fff"
            stroke="#1890ff"
            strokeWidth={2}
            className="cad-workstation-center-point"
            style={{ cursor: 'grab' }}
          />
        </>
      )}

      {/* 方向指示器（有旋转时显示） */}
      {rotation !== 0 && (
        <g style={{ pointerEvents: 'none' }}>
          <line
            x1={geometry.labelCenterX}
            y1={geometry.labelCenterY - 5}
            x2={geometry.labelCenterX}
            y2={geometry.labelCenterY - 20}
            stroke={lineColor}
            strokeWidth={1}
          />
          <polygon
            points={`${geometry.labelCenterX},${geometry.labelCenterY - 23} ${geometry.labelCenterX - 3},${geometry.labelCenterY - 16} ${geometry.labelCenterX + 3},${geometry.labelCenterY - 16}`}
            fill={lineColor}
          />
        </g>
      )}
    </g>
  );
};

export default WorkstationElement;


