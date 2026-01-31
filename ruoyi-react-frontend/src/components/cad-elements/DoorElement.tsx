/**
 * 门元素组件 - CAD 标准制图样式
 */

import { useMemo } from 'react';
import { getDoorColor } from '@/components/cad-editor/theme';
import { rotatePoint } from '@/utils/cad/geometry';
import type { Door } from '@/components/cad-editor/types';

export interface DoorElementProps {
  door: Door;
  selected?: boolean;
  hovered?: boolean;
  onSelect?: () => void;
  onHover?: (hovered: boolean) => void;
  onDoubleClick?: () => void;
  style?: React.CSSProperties;
}

export const DoorElement: React.FC<DoorElementProps> = ({
  door,
  selected = false,
  hovered = false,
  onSelect,
  onHover,
  onDoubleClick,
  style,
}) => {
  const color = useMemo(
    () => getDoorColor(door, selected, hovered),
    [door, selected, hovered]
  );

  // CAD 标准颜色
  const lineColor = '#2c3e50'; // 深灰色线条
  const highlightColor = selected ? '#1890ff' : hovered ? '#40a9ff' : '#3498db';

  // 计算门的位置和方向
  const doorGeometry = useMemo(() => {
    const { position, width, length, angle, direction, type } = door;

    // 门厚度（CAD 标准）
    const doorThickness = 5;

    // 门开口宽度
    const openingWidth = width;

    // 根据角度计算门洞位置
    const angleRad = (angle * Math.PI) / 180;
    const cos = Math.cos(angleRad);
    const sin = Math.sin(angleRad);

    // 门洞的起点和终点（墙体上的开口）
    // direction 表示门的开启方向：
    // - 'left'  = 门向左开  = 合页在右边（+方向）
    // - 'right' = 门向右开  = 合页在左边（-方向）
    const halfWidth = openingWidth / 2;
    const hingePoint = {
      x: direction === 'left' ? position.x + cos * halfWidth : position.x - cos * halfWidth,
      y: direction === 'left' ? position.y + sin * halfWidth : position.y - sin * halfWidth,
    };
    const openEndPoint = {
      x: direction === 'left' ? position.x - cos * halfWidth : position.x + cos * halfWidth,
      y: direction === 'left' ? position.y - sin * halfWidth : position.y + sin * halfWidth,
    };

    // 计算门扇（用弧线表示开启方向）
    let doorArcPath = '';
    let doorAngle = angle;

    switch (direction) {
      case 'left':
        // 门向左开（合页在右），门扇逆时针旋转90度
        doorAngle += 90;
        break;
      case 'right':
        // 门向右开（合页在左），门扇顺时针旋转90度
        doorAngle -= 90;
        break;
      case 'double':
        // 双开门不需要调整
        break;
      case 'sliding':
        // 推拉门用虚线表示
        break;
    }

    // 计算门扇终点
    const leafAngleRad = (doorAngle * Math.PI) / 180;
    const leafEndX = hingePoint.x + Math.cos(leafAngleRad) * length;
    const leafEndY = hingePoint.y + Math.sin(leafAngleRad) * length;

    // 双开门的第二扇
    let leafEndX2, leafEndY2;
    if (type === 'double') {
      leafEndX2 = openEndPoint.x + Math.cos(leafAngleRad + Math.PI) * length;
      leafEndY2 = openEndPoint.y + Math.sin(leafAngleRad + Math.PI) * length;
    }

    return { hingePoint, openEndPoint, leafEndX, leafEndY, leafEndX2, leafEndY2, angleRad, leafAngleRad };
  }, [door.position, door.width, door.length, door.angle, door.direction, door.type]);

  const handleMouseEnter = () => {
    onHover?.(true);
  };

  const handleMouseLeave = () => {
    onHover?.(false);
  };

  return (
    <g
      className={`cad-door ${selected ? 'cad-door-selected' : ''} ${hovered ? 'cad-door-hovered' : ''}`}
      onClick={onSelect}
      onDoubleClick={onDoubleClick}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      style={style}
    >
      {/* 门洞（墙体开口）- 绘制在墙体内，所以用白色填充遮挡线条 */}
      <line
        x1={doorGeometry.hingePoint.x}
        y1={doorGeometry.hingePoint.y}
        x2={doorGeometry.openEndPoint.x}
        y2={doorGeometry.openEndPoint.y}
        stroke="#ffffff"
        strokeWidth={12}
        style={{ pointerEvents: 'none' }}
      />

      {/* 门扇弧线 - CAD 标准表示法 */}
      {door.type === 'sliding' ? (
        // 推拉门 - 用虚线箭头表示
        <g style={{ pointerEvents: 'none' }}>
          <line
            x1={doorGeometry.hingePoint.x}
            y1={doorGeometry.hingePoint.y}
            x2={doorGeometry.leafEndX}
            y2={doorGeometry.leafEndY}
            stroke={highlightColor}
            strokeWidth={2}
            strokeDasharray="4,3"
          />
          <polygon
            points={`${doorGeometry.leafEndX},${doorGeometry.leafEndY} ${doorGeometry.leafEndX - 6},${doorGeometry.leafEndY + 3} ${doorGeometry.leafEndX - 6},${doorGeometry.leafEndY - 3}`}
            fill={highlightColor}
          />
        </g>
      ) : door.type === 'double' ? (
        // 双开门 - 两扇门弧线
        <g style={{ pointerEvents: 'none' }}>
          <path
            d={`M ${doorGeometry.hingePoint.x} ${doorGeometry.hingePoint.y}
                    A ${door.length} ${door.length} 0 0 ${door.direction === 'left' ? 1 : 0}
                    ${doorGeometry.leafEndX} ${doorGeometry.leafEndY}`}
            fill="none"
            stroke={highlightColor}
            strokeWidth={2}
          />
          <path
            d={`M ${doorGeometry.openEndPoint.x} ${doorGeometry.openEndPoint.y}
                    A ${door.length} ${door.length} 0 0 ${door.direction === 'left' ? 0 : 1}
                    ${doorGeometry.leafEndX2} ${doorGeometry.leafEndY2}`}
            fill="none"
            stroke={highlightColor}
            strokeWidth={2}
          />
        </g>
      ) : (
        // 单开门 - 门扇弧线
        <path
          d={`M ${doorGeometry.hingePoint.x} ${doorGeometry.hingePoint.y}
                  A ${door.length} ${door.length} 0 0 ${door.direction === 'left' ? 1 : 0}
                  ${doorGeometry.leafEndX} ${doorGeometry.leafEndY}`}
          fill="none"
          stroke={highlightColor}
          strokeWidth={2}
          style={{ pointerEvents: 'none' }}
        />
      )}

      {/* 门框线（CAD 标准：细线表示门框） */}
      <line
        x1={doorGeometry.hingePoint.x}
        y1={doorGeometry.hingePoint.y}
        x2={doorGeometry.openEndPoint.x}
        y2={doorGeometry.openEndPoint.y}
        stroke={lineColor}
        strokeWidth={1}
        style={{ pointerEvents: 'none' }}
      />

      {/* 合页标记 - CAD 标准符号 */}
      <circle
        cx={doorGeometry.hingePoint.x}
        cy={doorGeometry.hingePoint.y}
        r={2.5}
        fill={lineColor}
        style={{ pointerEvents: 'none' }}
      />

      {/* 选中状态的控制点 */}
      {selected && (
        <>
          <circle
            cx={door.position.x}
            cy={door.position.y}
            r={4}
            fill="#fff"
            stroke="#1890ff"
            strokeWidth={2}
            style={{ cursor: 'move' }}
          />
          <circle
            cx={doorGeometry.hingePoint.x}
            cy={doorGeometry.hingePoint.y}
            r={3}
            fill="#fff"
            stroke="#1890ff"
            strokeWidth={1.5}
            style={{ cursor: 'pointer' }}
          />
          <circle
            cx={doorGeometry.openEndPoint.x}
            cy={doorGeometry.openEndPoint.y}
            r={3}
            fill="#fff"
            stroke="#1890ff"
            strokeWidth={1.5}
            style={{ cursor: 'pointer' }}
          />
        </>
      )}

      {/* 门类型标识 - CAD 标准符号 */}
      {door.type === 'emergency' && (
        <text
          x={door.position.x}
          y={door.position.y - door.length / 2 - 12}
          fill="#e74c3c"
          fontSize={9}
          textAnchor="middle"
          fontWeight="bold"
          style={{ pointerEvents: 'none' }}
        >
          E
        </text>
      )}
    </g>
  );
};

export default DoorElement;
