/**
 * 墙体元素组件
 */

import { useMemo } from 'react';
import { getWallColor } from '@/components/cad-editor/theme';
import type { Wall } from '@/components/cad-editor/types';

export interface WallElementProps {
  wall: Wall;
  selected?: boolean;
  hovered?: boolean;
  onSelect?: () => void;
  onHover?: (hovered: boolean) => void;
  onDoubleClick?: () => void;
  style?: React.CSSProperties;
}

export const WallElement: React.FC<WallElementProps> = ({
  wall,
  selected = false,
  hovered = false,
  onSelect,
  onHover,
  onDoubleClick,
  style,
}) => {
  const color = useMemo(
    () => getWallColor(wall, selected, hovered),
    [wall, selected, hovered]
  );

  const pathData = useMemo(() => {
    if (wall.points.length === 0) return '';

    const first = wall.points[0];
    let path = `M ${first.x} ${first.y}`;

    for (let i = 1; i < wall.points.length; i++) {
      const point = wall.points[i];
      path += ` L ${point.x} ${point.y}`;
    }

    return path;
  }, [wall.points]);

  const handleMouseEnter = () => {
    onHover?.(true);
  };

  const handleMouseLeave = () => {
    onHover?.(false);
  };

  return (
    <g
      className={`cad-wall ${selected ? 'cad-wall-selected' : ''} ${hovered ? 'cad-wall-hovered' : ''}`}
      onClick={onSelect}
      onDoubleClick={onDoubleClick}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      style={style}
    >
      {/* 墙体轮廓 */}
      <path
        d={pathData}
        stroke={color}
        strokeWidth={wall.thickness}
        fill="none"
        strokeLinecap="square"
        strokeLinejoin="miter"
        style={{ cursor: 'pointer' }}
      />

      {/* 选中状态下的控制点 */}
      {selected && wall.points.length > 0 && (
        <>
          {wall.points.map((point, index) => (
            <circle
              key={index}
              cx={point.x}
              cy={point.y}
              r={5}
              fill="#fff"
              stroke={color}
              strokeWidth={2}
              className="cad-wall-control-point"
              style={{ cursor: 'move' }}
            />
          ))}
        </>
      )}
    </g>
  );
};

export default WallElement;
