/**
 * CAD 图层面板组件
 */

import { useCallback, useState } from "react";
import { Card, Slider, Space } from "antd";
import { EyeOutlined, EyeInvisibleOutlined, LockOutlined, UnlockOutlined } from "@ant-design/icons";
import type { Layer } from "./types";
import "./CADLayersPanel.css";

export interface CADLayersPanelProps {
  layers: Layer[];
  onLayerVisibilityChange: (layerId: string, visible: boolean) => void;
  onLayerLockChange: (layerId: string, locked: boolean) => void;
  onLayerOpacityChange?: (layerId: string, opacity: number) => void;
  style?: React.CSSProperties;
}

interface LayerItemProps {
  layer: Layer;
  onVisibilityChange: (visible: boolean) => void;
  onLockChange: (locked: boolean) => void;
  onOpacityChange?: (opacity: number) => void;
}

const LayerItem = function LayerItem({
  layer,
  onVisibilityChange,
  onLockChange,
  onOpacityChange,
}: LayerItemProps) {
  const [localOpacity, setLocalOpacity] = useState(layer.opacity);

  const handleOpacityChange = (value: number) => {
    setLocalOpacity(value);
    onOpacityChange?.(value);
  };

  return (
    <div className="cad-layer-item">
      {/* 第一行：名称和图标按钮 */}
      <div className="cad-layer-row">
        <span className="cad-layer-name">{layer.name}</span>
        <div className="cad-layer-actions">
          <button
            className={`cad-layer-icon-btn ${!layer.visible ? "cad-layer-icon-btn-hidden" : ""}`}
            onClick={() => onVisibilityChange(!layer.visible)}
            title={layer.visible ? "隐藏" : "显示"}
          >
            {layer.visible ? <EyeOutlined /> : <EyeInvisibleOutlined />}
          </button>
          <button
            className="cad-layer-icon-btn"
            onClick={() => onLockChange(!layer.locked)}
            title={layer.locked ? "解锁" : "锁定"}
          >
            {layer.locked ? <LockOutlined /> : <UnlockOutlined />}
          </button>
        </div>
      </div>
      {/* 第二行：透明度滑块 */}
      {onOpacityChange && (
        <div className="cad-layer-row cad-layer-opacity-row">
          <span className="cad-layer-opacity-label">透明度</span>
          <Slider
            min={0}
            max={1}
            step={0.1}
            value={localOpacity}
            onChange={handleOpacityChange}
            className="cad-layer-opacity-slider"
          />
        </div>
      )}
    </div>
  );
};

export const CADLayersPanel = function CADLayersPanel({
  layers,
  onLayerVisibilityChange,
  onLayerLockChange,
  onLayerOpacityChange,
  style,
}: CADLayersPanelProps) {
  const handleVisibilityChange = useCallback(
    (layerId: string, visible: boolean) => {
      onLayerVisibilityChange(layerId, visible);
    },
    [onLayerVisibilityChange]
  );

  const handleLockChange = useCallback(
    (layerId: string, locked: boolean) => {
      onLayerLockChange(layerId, locked);
    },
    [onLayerLockChange]
  );

  const handleOpacityChange = useCallback(
    (layerId: string, opacity: number) => {
      onLayerOpacityChange?.(layerId, opacity);
    },
    [onLayerOpacityChange]
  );

  return (
    <Card
      title="图层"
      size="small"
      style={{ width: 180, ...style }}
      styles={{ body: { padding: 8 } }}
    >
      <Space orientation="vertical" size={6} style={{ width: "100%" }}>
        {layers.map((layer) => (
          <LayerItem
            key={layer.id}
            layer={layer}
            onVisibilityChange={(visible) => handleVisibilityChange(layer.id, visible)}
            onLockChange={(locked) => handleLockChange(layer.id, locked)}
            onOpacityChange={
              onLayerOpacityChange ? (opacity) => handleOpacityChange(layer.id, opacity) : undefined
            }
          />
        ))}
      </Space>
    </Card>
  );
};

export default CADLayersPanel;
