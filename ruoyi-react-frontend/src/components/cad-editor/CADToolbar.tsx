/**
 * CAD 工具栏组件
 */

import { useCallback, useMemo } from "react";
import { Button, Space, Tooltip } from "antd";
import {
  BorderOutlined,
  ApartmentOutlined,
  DesktopOutlined,
  FontSizeOutlined,
  ZoomInOutlined,
  ZoomOutOutlined,
  CompressOutlined,
  SaveOutlined,
  UndoOutlined,
  RedoOutlined,
} from "@ant-design/icons";
import type { EditorMode } from "./types";

// 工具按钮配置
interface ToolButtonConfig {
  key: EditorMode;
  icon: React.ReactNode;
  label: string;
}

const DRAWING_TOOLS: readonly ToolButtonConfig[] = [
  { key: "draw_wall", icon: <BorderOutlined />, label: "墙体" },
  { key: "draw_door", icon: <ApartmentOutlined />, label: "门" },
  { key: "draw_workstation", icon: <DesktopOutlined />, label: "工位" },
  { key: "draw_text", icon: <FontSizeOutlined />, label: "文本" },
] as const;

export interface CADToolbarProps {
  mode: EditorMode;
  onModeChange: (mode: EditorMode) => void;
  onZoomIn: () => void;
  onZoomOut: () => void;
  onResetView: () => void;
  onSave: () => void;
  canUndo: boolean;
  canRedo: boolean;
  onUndo: () => void;
  onRedo: () => void;
  readOnly?: boolean;
  style?: React.CSSProperties;
}

export function CADToolbar({
  mode,
  onModeChange,
  onZoomIn,
  onZoomOut,
  onResetView,
  onSave,
  canUndo,
  canRedo,
  onUndo,
  onRedo,
  readOnly = false,
  style,
}: CADToolbarProps) {
  const handleModeToggle = useCallback(
    (toolMode: EditorMode) => {
      onModeChange(mode === toolMode ? "select" : toolMode);
    },
    [mode, onModeChange]
  );

  const toolButtons = useMemo(
    () =>
      DRAWING_TOOLS.map((tool) => ({
        ...tool,
        disabled: readOnly,
      })),
    [readOnly]
  );

  const renderToolButton = function (tool: ToolButtonConfig & { disabled: boolean }) {
    const isActive = mode === tool.key;

    return (
      <Tooltip key={tool.key} title={tool.label}>
        <Button
          type={isActive ? "primary" : "default"}
          icon={tool.icon}
          disabled={tool.disabled}
          onClick={() => handleModeToggle(tool.key)}
        >
          {tool.label}
        </Button>
      </Tooltip>
    );
  };

  const toolbarStyle: React.CSSProperties = {
    display: "flex",
    alignItems: "center",
    gap: 8,
    padding: "8px 16px",
    background: "var(--theme-text-inverse, #fff)",
    borderBottom: "1px solid var(--theme-border-divider, #e8e8e8)",
    ...style,
  };

  const dividerStyle: React.CSSProperties = {
    width: 1,
    height: 24,
    background: "var(--theme-border-divider, #e8e8e8)",
  };

  return (
    <div className="cad-toolbar" style={toolbarStyle}>
      <Space size="small">{toolButtons.map(renderToolButton)}</Space>

      <div style={{ flex: 1 }} />

      <Space size="small">
        <Tooltip title="放大">
          <Button icon={<ZoomInOutlined />} onClick={onZoomIn} />
        </Tooltip>
        <Tooltip title="缩小">
          <Button icon={<ZoomOutOutlined />} onClick={onZoomOut} />
        </Tooltip>
        <Tooltip title="重置视图">
          <Button icon={<CompressOutlined />} onClick={onResetView} />
        </Tooltip>
      </Space>

      <div style={dividerStyle} />

      <Space size="small">
        <Tooltip title="撤销 (Ctrl+Z)">
          <Button icon={<UndoOutlined />} disabled={!canUndo} onClick={onUndo} />
        </Tooltip>
        <Tooltip title="重做 (Ctrl+Y)">
          <Button icon={<RedoOutlined />} disabled={!canRedo} onClick={onRedo} />
        </Tooltip>
      </Space>

      <div style={dividerStyle} />

      <Tooltip title="保存 (Ctrl+S)">
        <Button type="primary" icon={<SaveOutlined />} disabled={readOnly} onClick={onSave}>
          保存
        </Button>
      </Tooltip>
    </div>
  );
}
