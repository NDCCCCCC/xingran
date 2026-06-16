/**
 * 文本元素组件
 */

import { useMemo, useCallback } from "react";
import type { TextElement } from "@/components/cad-editor/types";

export interface TextElementProps {
  text: TextElement;
  selected?: boolean;
  hovered?: boolean;
  onSelect?: () => void;
  onHover?: (hovered: boolean) => void;
  onDoubleClick?: () => void;
  style?: React.CSSProperties;
}

const DEFAULT_FONT_SIZE = 14;
const DEFAULT_FONT_COLOR = "var(--theme-text-primary, #333333)";
const DEFAULT_FONT_FAMILY = "Arial, sans-serif";
const SELECTION_BG_COLOR = "rgba(24, 144, 255, 0.1)";
const SELECTION_BORDER_COLOR = "#1890ff";
const CONTROL_POINT_RADIUS = 4;
const CONTROL_POINT_STROKE_WIDTH = 2;

export function CADTextElement({
  text,
  selected = false,
  hovered = false,
  onSelect,
  onHover,
  onDoubleClick,
  style,
}: TextElementProps) {
  const fontSize = text.fontSize ?? DEFAULT_FONT_SIZE;
  const textColor = text.color ?? DEFAULT_FONT_COLOR;

  const textStyle = useMemo(() => ({
    fill: textColor,
    fontSize,
    fontFamily: text.fontFamily ?? DEFAULT_FONT_FAMILY,
    fontWeight: text.fontWeight ?? "normal",
    fontStyle: text.fontStyle ?? "normal",
    cursor: "pointer",
    userSelect: "none" as const,
  }), [textColor, fontSize, text.fontFamily, text.fontWeight, text.fontStyle]);

  const transform = useMemo(() => {
    if (!text.angle) return "";
    return `rotate(${text.angle} ${text.position.x} ${text.position.y})`;
  }, [text.angle, text.position]);

  const handleMouseEnter = useCallback(() => {
    onHover?.(true);
  }, [onHover]);

  const handleMouseLeave = useCallback(() => {
    onHover?.(false);
  }, [onHover]);

  const containerClassName = [
    "cad-text",
    selected && "cad-text-selected",
    hovered && "cad-text-hovered",
  ].filter(Boolean).join(" ");

  // 选中背景的尺寸计算
  const selectionBg = useMemo(() => {
    if (!selected) return null;

    const textWidth = text.content.length * fontSize * 0.6;
    const padding = 5;
    const height = fontSize * 1.2;

    return {
      x: text.position.x - padding,
      y: text.position.y - fontSize,
      width: textWidth + padding * 2,
      height,
    };
  }, [selected, text.content, text.position, fontSize]);

  return (
    <g
      className={containerClassName}
      onClick={onSelect}
      onDoubleClick={onDoubleClick}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      style={style}
      transform={transform}
    >
      {selectionBg && (
        <rect
          x={selectionBg.x}
          y={selectionBg.y}
          width={selectionBg.width}
          height={selectionBg.height}
          fill={SELECTION_BG_COLOR}
          stroke={SELECTION_BORDER_COLOR}
          strokeWidth={1}
          strokeDasharray="3,3"
          style={{ pointerEvents: "none" }}
        />
      )}

      <text
        x={text.position.x}
        y={text.position.y}
        style={textStyle}
        textAnchor="start"
        dominantBaseline="hanging"
      >
        {text.content}
      </text>

      {selected && (
        <circle
          cx={text.position.x}
          cy={text.position.y}
          r={CONTROL_POINT_RADIUS}
          fill="#fff"
          stroke={SELECTION_BORDER_COLOR}
          strokeWidth={CONTROL_POINT_STROKE_WIDTH}
          style={{ cursor: "move" }}
        />
      )}
    </g>
  );
}
