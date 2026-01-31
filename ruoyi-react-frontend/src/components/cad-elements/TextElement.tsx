/**
 * 文本元素组件
 */

import { useMemo } from 'react';
import type { TextElement } from '@/components/cad-editor/types';

export interface TextElementProps {
  text: TextElement;
  selected?: boolean;
  hovered?: boolean;
  onSelect?: () => void;
  onHover?: (hovered: boolean) => void;
  onDoubleClick?: () => void;
  style?: React.CSSProperties;
}

export const CADTextElement: React.FC<TextElementProps> = ({
  text,
  selected = false,
  hovered = false,
  onSelect,
  onHover,
  onDoubleClick,
  style,
}) => {
  const textStyle = useMemo(() => ({
    fill: text.color || '#333333',
    fontSize: text.fontSize || 14,
    fontFamily: text.fontFamily || 'Arial, sans-serif',
    fontWeight: text.fontWeight || 'normal',
    fontStyle: text.fontStyle || 'normal',
    cursor: 'pointer',
    userSelect: 'none',
  }), [text]);

  const transform = useMemo(() => {
    const transforms = [];
    if (text.angle) {
      transforms.push(`rotate(${text.angle} ${text.position.x} ${text.position.y})`);
    }
    return transforms.join(' ');
  }, [text.angle, text.position]);

  const handleMouseEnter = () => {
    onHover?.(true);
  };

  const handleMouseLeave = () => {
    onHover?.(false);
  };

  return (
    <g
      className={`cad-text ${selected ? 'cad-text-selected' : ''} ${hovered ? 'cad-text-hovered' : ''}`}
      onClick={onSelect}
      onDoubleClick={onDoubleClick}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      style={style}
      transform={transform}
    >
      {/* 选中状态背景 */}
      {selected && (
        <rect
          x={text.position.x - 5}
          y={text.position.y - (text.fontSize || 14)}
          width={text.content.length * (text.fontSize || 14) * 0.6 + 10}
          height={(text.fontSize || 14) * 1.2}
          fill="rgba(24, 144, 255, 0.1)"
          stroke="#1890ff"
          strokeWidth={1}
          strokeDasharray="3,3"
          style={{ pointerEvents: 'none' }}
        />
      )}

      {/* 文本内容 */}
      <text
        x={text.position.x}
        y={text.position.y}
        style={textStyle}
        textAnchor="start"
        dominantBaseline="hanging"
      >
        {text.content}
      </text>

      {/* 选中状态的控制点 */}
      {selected && (
        <circle
          cx={text.position.x}
          cy={text.position.y}
          r={4}
          fill="#fff"
          stroke="#1890ff"
          strokeWidth={2}
          style={{ cursor: 'move' }}
        />
      )}
    </g>
  );
};

export default CADTextElement;
