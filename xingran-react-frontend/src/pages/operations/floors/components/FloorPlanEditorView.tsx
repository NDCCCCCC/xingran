import type { FC } from "react";
import { Card, Button, Space, Spin, Select } from "antd";
import {
  EyeOutlined,
  EditOutlined,
  ReloadOutlined,
  ArrowLeftOutlined,
  BuildOutlined,
  AppstoreOutlined,
} from "@ant-design/icons";
import type { Floor, Building } from "@/types";
import { CADFloorPlanEditor } from "@/components/cad-editor";
import type { FloorPlanData } from "@/components/cad-editor/types";
import {
  DepartmentTreeSelect,
  type TreeNode,
  type Department,
} from "@/components/shared/DepartmentTreeSelect";

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
  departmentTreeData?: TreeNode[];
  departments?: Department[];
  buildingOptions: Building[];
  floorOptions: FloorOption[];
  selectedDeptId: string;
  selectedDeptName?: string;
  selectedBuildingId: string;
  selectedBuildingName?: string;
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
  departments,
  buildingOptions,
  floorOptions,
  selectedDeptId,
  selectedDeptName,
  selectedBuildingId,
  selectedBuildingName,
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
          display: "flex",
          justifyContent: "center",
          alignItems: "center",
          height: "60vh",
          flexDirection: "column",
          gap: 16,
        }}
      >
        <Spin size="large" />
        <span style={{ color: "var(--theme-text-tertiary, #999)" }}>加载中...</span>
      </div>
    );
  }

  return (
    <div style={{ height: "calc(100vh - 180px)", display: "flex", flexDirection: "column" }}>
      <Card
        title={`${selectedDeptName || "机构"} - ${selectedBuildingName || currentFloor?.buildingName || "楼宇"} - ${currentFloor.name || `${currentFloor.floorNo}层`} - 平面图编辑`}
        extra={
          <Space size="middle">
            <Space size="small" style={{ display: "flex", alignItems: "center" }}>
              <BuildOutlined style={{ color: "var(--theme-info, #1890ff)" }} />
              <DepartmentTreeSelect
                style={{ width: 180 }}
                placeholder="选择机构"
                treeData={departmentTreeData}
                departments={departments}
                value={selectedDeptId}
                onChange={onDepartmentChange}
                allowClear={false}
              />
            </Space>

            <Space size="small" style={{ display: "flex", alignItems: "center" }}>
              <AppstoreOutlined style={{ color: "var(--theme-purple, #722ed1)" }} />
              <Select
                style={{ width: 150 }}
                placeholder="选择楼宇"
                value={selectedBuildingId}
                onChange={onBuildingChange}
                options={buildingOptions.map((b) => ({
                  label: b.name,
                  value: b.id,
                }))}
                showSearch
                optionFilterProp="label"
                onSearch={() => {}}
              />
            </Space>

            <Space size="small" style={{ display: "flex", alignItems: "center" }}>
              <AppstoreOutlined style={{ color: "var(--theme-success, #52c41a)" }} />
              <Select
                style={{ width: 150 }}
                placeholder="选择楼层"
                value={currentFloor?.id}
                onChange={onFloorChange}
                options={floorOptions.map((f) => ({
                  label: f.name || `${f.floorNo}层`,
                  value: f.id,
                }))}
                showSearch
                optionFilterProp="label"
                onSearch={() => {}}
              />
            </Space>

            <Button
              icon={isEditMode ? <EyeOutlined /> : <EditOutlined />}
              onClick={() => onEditModeChange(!isEditMode)}
            >
              {isEditMode ? "预览模式" : "编辑模式"}
            </Button>
            <Button icon={<ReloadOutlined />} onClick={onRefresh} loading={loading}>
              刷新
            </Button>
            <Button icon={<ArrowLeftOutlined />} onClick={onBack}>
              返回列表
            </Button>
          </Space>
        }
        style={{ flex: 1, display: "flex", flexDirection: "column" }}
        styles={{ body: { flex: 1, padding: 0, overflow: "hidden" } }}
      >
        <CADFloorPlanEditor
          floorId={currentFloor.id}
          floorName={currentFloor.name || `${currentFloor.floorNo}层`}
          walls={floorPlanData.walls}
          doors={floorPlanData.doors}
          workstations={floorPlanData.workstations}
          texts={floorPlanData.texts || []}
          planImageId={currentFloor.planImageId}
          planImageUrl={currentFloor.planImageUrl}
          layerConfig={floorPlanData.layerConfig}
          onSave={onSave}
          readOnly={!isEditMode}
        />
      </Card>
    </div>
  );
};
