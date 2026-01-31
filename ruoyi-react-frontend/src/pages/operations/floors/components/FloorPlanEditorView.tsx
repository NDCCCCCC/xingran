/**
 * 楼层平面图编辑器视图组件
 */

import type { FC } from 'react';
import { Card, Button, Space, Spin } from 'antd';
import {
  EyeOutlined,
  EditOutlined,
  ReloadOutlined,
  ArrowLeftOutlined,
} from '@ant-design/icons';
import type { Floor } from '@/types';
import { CADFloorPlanEditor } from '@/components/cad-editor';
import type { FloorPlanData } from '@/components/cad-editor/types';

interface FloorPlanEditorViewProps {
  currentFloor: Floor | null;
  floorPlanData: FloorPlanData | null;
  loading: boolean;
  isEditMode: boolean;
  onEditModeChange: (edit: boolean) => void;
  onRefresh: () => void;
  onBack: () => void;
  onSave: (data: FloorPlanData) => Promise<void>;
}

export const FloorPlanEditorView: FC<FloorPlanEditorViewProps> = ({
  currentFloor,
  floorPlanData,
  loading,
  isEditMode,
  onEditModeChange,
  onRefresh,
  onBack,
  onSave,
}) => {
  if (!currentFloor || !floorPlanData) {
    return (
      <div
        style={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          height: '60vh',
          flexDirection: 'column',
          gap: 16,
        }}
      >
        <Spin size="large" />
        <span style={{ color: '#999' }}>加载中...</span>
      </div>
    );
  }

  return (
    <div style={{ height: 'calc(100vh - 180px)', display: 'flex', flexDirection: 'column' }}>
      <Card
        title={`${currentFloor.name || `${currentFloor.floorNo}层`} - 平面图编辑`}
        extra={
          <Space>
            <Button
              icon={isEditMode ? <EyeOutlined /> : <EditOutlined />}
              onClick={() => onEditModeChange(!isEditMode)}
            >
              {isEditMode ? '预览模式' : '编辑模式'}
            </Button>
            <Button icon={<ReloadOutlined />} onClick={onRefresh} loading={loading}>
              刷新
            </Button>
            <Button icon={<ArrowLeftOutlined />} onClick={onBack}>
              返回列表
            </Button>
          </Space>
        }
        style={{ flex: 1, display: 'flex', flexDirection: 'column' }}
        styles={{ body: { flex: 1, padding: 0, overflow: 'hidden' } }}
      >
        <CADFloorPlanEditor
          floorId={currentFloor.id}
          floorName={currentFloor.name || `${currentFloor.floorNo}层`}
          walls={floorPlanData.walls}
          doors={floorPlanData.doors}
          workstations={floorPlanData.workstations}
          texts={floorPlanData.texts || []}
          onSave={onSave}
          readOnly={!isEditMode}
        />
      </Card>
    </div>
  );
};
