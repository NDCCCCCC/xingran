import type { FC } from "react";
import { Card, Button, Space, Spin } from "antd";
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
import type { ActionPermissionDecision } from "@/components/shared";
import {
  DepartmentTreeSelect,
  type TreeNode,
  type Department,
} from "@/components/shared/DepartmentTreeSelect";
// Phase 71 (W3)：裸 Select → AppSelect（本文件两处已是 options 形态，直接迁）。
import { AppSelect } from "@/components/ui";

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
  /**
   * Phase 86 W2：平面图保存/编辑的**四码合并决策**
   * （`opsFloorPlan.edit ∧ opsWall.* ∧ opsDoor.* ∧ opsWorkstation.edit`）。
   * `visible=false` → 不渲染「编辑模式」按钮（ready 缺任一码）；
   * `disabled=true` → 渲染但禁用（未就绪）。不传 = 不做判定（既有调用方行为零变更）。
   */
  editDecision?: ActionPermissionDecision;
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
  editDecision,
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
        <span style={{ color: "var(--theme-text-tertiary)" }}>加载中...</span>
      </div>
    );
  }

  return (
    <div style={{ height: "calc(100vh - 180px)", display: "flex", flexDirection: "column" }}>
      <Card
        title={`${selectedDeptName || "机构"} - ${selectedBuildingName || currentFloor?.buildingName || "楼宇"} - ${currentFloor.name || `${currentFloor.floorNo}层`} - 平面图编辑`}
        extra={
          <Space size="medium">
            <Space size="small" style={{ display: "flex", alignItems: "center" }}>
              <BuildOutlined style={{ color: "var(--theme-info)" }} />
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
              <AppstoreOutlined style={{ color: "var(--theme-purple)" }} />
              <AppSelect
                allowClear
                style={{ width: 150 }}
                placeholder="选择楼宇"
                value={selectedBuildingId}
                // AppSelect 的 onChange 值为 unknown；父级 handleEditorBuildingChange 对空值
                // 直接 return，这里收敛成 "" 保持「未选择」语义，不引入新分支
                onChange={(value) => onBuildingChange((value as string) ?? "")}
                options={buildingOptions.map((b) => ({
                  label: b.name,
                  value: b.id,
                }))}
                showSearch
                onSearch={() => {}}
              />
            </Space>

            <Space size="small" style={{ display: "flex", alignItems: "center" }}>
              <AppstoreOutlined style={{ color: "var(--theme-success)" }} />
              <AppSelect
                allowClear
                style={{ width: 150 }}
                placeholder="选择楼层"
                value={currentFloor?.id}
                // 同上：清空收敛为 ""（父级 handleFloorChange 内部对 "" 自会处理空态）
                onChange={(value) => onFloorChange((value as string) ?? "")}
                options={floorOptions.map((f) => ({
                  label: f.name || `${f.floorNo}层`,
                  value: f.id,
                }))}
                showSearch
                onSearch={() => {}}
              />
            </Space>

            {/* Phase 86 W2：平面图编辑/保存入口 —— 四码合并决策。
                ready 缺任一码 → 不渲染（该入口直通可保存的编辑器）；
                未就绪 → 渲染但 disabled + hint（三态语义）。
                不传 editDecision 时保持既有行为（恒可见可用）。 */}
            {(editDecision?.visible ?? true) && (
              <Button
                icon={isEditMode ? <EyeOutlined /> : <EditOutlined />}
                disabled={editDecision?.disabled}
                title={editDecision?.hint}
                onClick={() => onEditModeChange(!isEditMode)}
              >
                {isEditMode ? "预览模式" : "编辑模式"}
              </Button>
            )}
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
