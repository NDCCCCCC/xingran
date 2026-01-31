/**
 * CAD 工具栏组件
 */

import { useCallback, useMemo } from 'react';
import { Button, Space, Tooltip } from 'antd';
import type { ButtonProps } from 'antd';
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
} from '@ant-design/icons';
import type { ToolAction, EditorMode } from './types';

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

interface ToolButton {
  key: ToolAction;
  icon: React.ReactNode;
  label: string;
  disabled?: boolean;
}

export const CADToolbar = function CADToolbar({
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
  const handleModeChange = useCallback(
    (newMode: EditorMode) => {
      onModeChange(newMode === mode ? 'select' : newMode);
    },
    [mode, onModeChange]
  );

  const toolButtons = useMemo<ToolButton[]>(
    () => [
      {
        key: 'draw_wall',
        icon: <BorderOutlined />,
        label: '墙体',
        disabled: readOnly,
      },
      {
        key: 'draw_door',
        icon: <ApartmentOutlined />,
        label: '门',
        disabled: readOnly,
      },
      {
        key: 'draw_workstation',
        icon: <DesktopOutlined />,
        label: '工位',
        disabled: readOnly,
      },
      {
        key: 'draw_text',
        icon: <FontSizeOutlined />,
        label: '文本',
        disabled: readOnly,
      },
    ],
    [readOnly]
  );

  const renderToolButton = function renderToolButton(tool: ToolButton) {
    const isActive = mode === tool.key;
    const buttonType: ButtonProps['type'] = isActive ? 'primary' : 'default';

    return (
      <Tooltip key={tool.key} title={tool.label}>
        <Button
          type={buttonType}
          icon={tool.icon}
          disabled={tool.disabled}
          onClick={() => handleModeChange(tool.key as EditorMode)}
        >
          {tool.label}
        </Button>
      </Tooltip>
    );
  };

  return (
    <div
      className="cad-toolbar"
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        padding: '8px 16px',
        background: '#fff',
        borderBottom: '1px solid #e8e8e8',
        ...style,
      }}
    >
      <Space size="small">
        {toolButtons.map(renderToolButton)}
      </Space>

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

      <div style={{ width: 1, height: 24, background: '#e8e8e8' }} />

      <Space size="small">
        <Tooltip title="撤销 (Ctrl+Z)">
          <Button icon={<UndoOutlined />} disabled={!canUndo} onClick={onUndo} />
        </Tooltip>
        <Tooltip title="重做 (Ctrl+Y)">
          <Button icon={<RedoOutlined />} disabled={!canRedo} onClick={onRedo} />
        </Tooltip>
      </Space>

      <div style={{ width: 1, height: 24, background: '#e8e8e8' }} />

      <Tooltip title="保存 (Ctrl+S)">
        <Button
          type="primary"
          icon={<SaveOutlined />}
          disabled={readOnly}
          onClick={onSave}
        >
          保存
        </Button>
      </Tooltip>
    </div>
  );
};

export default CADToolbar;
