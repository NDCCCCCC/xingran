/**
 * 墙体元素组件
 */

import { useMemo, useCallback } from "react";
import { getWallColor } from "@/components/cad-editor/theme";
import type { Wall } from "@/components/cad-editor/types";

export interface WallElementProps {
  wall: Wall;
  selected?: boolean;
  hovered?: boolean;
  onSelect?: () => void;
  onHover?: (hovered: boolean) => void;
  onDoubleClick?: () => void;
  style?: React.CSSProperties;
}

const CONTROL_POINT_RADIUS = 5;
const CONTROL_POINT_STROKE_WIDTH = 2;
const CONTROL_POINT_FILL = "#fff";

export function WallElement({
  wall,
  selected = false,
  hovered = false,
  onSelect,
  onHover,
  onDoubleClick,
  style,
}: WallElementProps) {
  const color = useMemo(
    () => getWallColor(wall, selected, hovered),
    [wall, selected, hovered]
  );

  const pathData = useMemo(() => {
    if (wall.points.length === 0) return "";

    const points = wall.points;
    const path = `M ${points[0].x} ${points[0].y}` +
      points.slice(1).map(p => ` L ${p.x} ${p.y}`).join("");

    return path;
  }, [wall.points]);

  const handleMouseEnter = useCallback(() => {
    onHover?.(true);
  }, [onHover]);

  const handleMouseLeave = useCallback(() => {
    onHover?.(false);
  }, [onHover]);

  const containerClassName = [
    "cad-wall",
    selected && "cad-wall-selected",
    hovered && "cad-wall-hovered",
  ].filter(Boolean).join(" ");

  return (
    <g
      className={containerClassName}
      onClick={onSelect}
      onDoubleClick={onDoubleClick}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      style={style}
    >
      <path
        d={pathData}
        stroke={color}
        strokeWidth={wall.thickness}
        fill="none"
        strokeLinecap="square"
        strokeLinejoin="miter"
        style={{ cursor: "pointer" }}
      />

      {selected && wall.points.length > 0 && (
        <>
          {wall.points.map((point, index) => (
            <circle
              key={index}
              cx={point.x}
              cy={point.y}
              r={CONTROL_POINT_RADIUS}
              fill={CONTROL_POINT_FILL}
              stroke={color}
              strokeWidth={CONTROL_POINT_STROKE_WIDTH}
              className="cad-wall-control-point"
              style={{ cursor: "move" }}
            />
          ))}
        </>
      )}
    </g>
  );
}
