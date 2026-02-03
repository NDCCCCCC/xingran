/**
 * 楼层平面图编辑器视图组件
 */

import type { FC } from 'react';
import { Card, Button, Space, Spin, Select } from 'antd';
import {
  EyeOutlined,
  EditOutlined,
  ReloadOutlined,
  ArrowLeftOutlined,
  BuildOutlined,
  AppstoreOutlined,
} from '@ant-design/icons';
import type { Floor, Building } from '@/types';
import { CADFloorPlanEditor } from '@/components/cad-editor';
import type { FloorPlanData } from '@/components/cad-editor/types';
import { DepartmentTreeSelect } from '@/components/shared/DepartmentTreeSelect';
import type { TreeNode } from '@/components/shared/DepartmentTreeSelect';

export interface FloorOption {
  id: string;
  name: string;
  floorNo: number;
}

interface FloorPlanEditorViewProps {
  currentFloor: Floor | null;
  floorPlanData: FloorPlanData | null;
  loading: boolean;
  isEditMode: boolean;
  departmentTreeData: TreeNode[];  // 外部机构部门树数据
  buildingOptions: Building[];     // 当前部门的楼宇选项
  floorOptions: FloorOption[];
  selectedDeptId: string;          // 当前选中的部门ID
  selectedBuildingId: string;      // 当前选中的楼宇ID
  onEditModeChange: (edit: boolean) => void;
  onRefresh: () => void;
  onBack: () => void;
  onSave: (data: FloorPlanData) => Promise<void>;
  onDepartmentChange: (deptId: string) => void;
  onBuildingChange: (buildingId: string) => void;
  onFloorChange: (floorId: string) => void;
}

export const FloorPlanEditorView: FC<FloorPlanEditorViewProps> = ({
  currentFloor,
  floorPlanData,
  loading,
  isEditMode,
  departmentTreeData,
  buildingOptions,
  floorOptions,
  selectedDeptId,
  selectedBuildingId,
  onEditModeChange,
  onRefresh,
  onBack,
  onSave,
  onDepartmentChange,
  onBuildingChange,
  onFloorChange,
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
          <Space size="middle">
            {/* 部门选择器（外部机构） */}
            <Space size="small" style={{ display: 'flex', alignItems: 'center' }}>
              <BuildOutlined style={{ color: '#1890ff' }} />
              <DepartmentTreeSelect
                style={{ width: 180 }}
                placeholder="选择机构"
                treeData={departmentTreeData}
                value={selectedDeptId}
                onChange={onDepartmentChange}
                allowClear={false}
              />
            </Space>

            {/* 楼宇选择器 */}
            <Space size="small" style={{ display: 'flex', alignItems: 'center' }}>
              <AppstoreOutlined style={{ color: '#722ed1' }} />
              <Select
                style={{ width: 150 }}
                placeholder="选择楼宇"
                value={selectedBuildingId}
                onChange={onBuildingChange}
                options={buildingOptions.map(b => ({
                  label: b.name,
                  value: b.id,
                }))}
                showSearch
                optionFilterProp="label"
              />
            </Space>

            {/* 楼层选择器 */}
            <Space size="small" style={{ display: 'flex', alignItems: 'center' }}>
              <AppstoreOutlined style={{ color: '#52c41a' }} />
              <Select
                style={{ width: 150 }}
                placeholder="选择楼层"
                value={currentFloor?.id}
                onChange={onFloorChange}
                options={floorOptions.map(f => ({
                  label: f.name || `${f.floorNo}层`,
                  value: f.id,
                }))}
                showSearch
                optionFilterProp="label"
              />
            </Space>

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
