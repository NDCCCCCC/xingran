/**
 * 楼层搜索表单组件
 */

import type { FC } from "react";
import { Form, Button, Space, Radio, Card } from "antd";
import {
  SearchOutlined,
  ReloadOutlined,
  TableOutlined,
  AppstoreOutlined,
  ImportOutlined,
  ExportOutlined,
  PlusOutlined,
  DeleteOutlined,
} from "@ant-design/icons";
import type { FormInstance } from "antd/es/form";
import type { ViewMode } from "../constants";
import { STATUS_OPTIONS } from "../constants";
import type { Building } from "@/types";
// Phase 71 (W3)：裸 Input/Select → 契约组件。
// 楼宇下拉原 children + showSearch（optionFilterProp="children"）→ options 改写（label 存在）；
// 状态枚举业务禁搜 → 显式 showSearch={false}。
import { AppInput, AppSelect } from "@/components/ui";

// 楼宇选项基础类型（只需要 id 和 name）
type BuildingOptionBase = Pick<Building, "id" | "name">;

interface FloorSearchFormProps {
  form: FormInstance;
  buildingOptions: BuildingOptionBase[];
  viewMode: ViewMode;
  loading: boolean;
  buildingOptionsByDept: BuildingOptionBase[];
  selectedDeptId: string;
  disabled?: boolean;
  onSearch: () => void;
  onReset: () => void;
  onRefresh: () => void;
  onViewModeChange: (mode: ViewMode) => void;
  onImport: () => void;
  onExport: () => void;
  onBatchDelete: () => void;
  onAdd: () => void;
  onBuildingChange?: (buildingId: string) => void;
  selectedCount: number;
}

export const FloorSearchForm: FC<FloorSearchFormProps> = ({
  form,
  buildingOptions: _buildingOptions,
  viewMode,
  loading,
  buildingOptionsByDept,
  selectedDeptId,
  disabled = false,
  onSearch,
  onReset,
  onRefresh,
  onViewModeChange,
  onImport,
  onExport,
  onBatchDelete,
  onAdd,
  onBuildingChange,
  selectedCount,
}) => {
  return (
    <Card style={{ marginBottom: 16 }}>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "flex-start",
          flexWrap: "wrap",
          gap: "16px",
        }}
      >
        <Form form={form} layout="inline" style={{ flex: 1, minWidth: 0 }} disabled={disabled}>
          <Form.Item name="buildingId" label="所属楼宇">
            {/* 形态分类器②③：children + 复合子节点 → options 改写；楼宇 UUID 字段 */}
            <AppSelect
              placeholder="请先选择部门"
              allowClear
              className="user-form-input"
              style={{ width: 150 }}
              showSearch
              disabled={disabled || !selectedDeptId}
              // AppSelect 的 onChange 值为 unknown；清空（null）时收敛成 "" 表示「未选楼宇」
              onChange={(value) => onBuildingChange?.((value as string) ?? "")}
              options={buildingOptionsByDept.map((b) => ({ value: b.id, label: b.name }))}
            />
          </Form.Item>

          <Form.Item name="floorNo" label="楼层号">
            <AppInput
              placeholder="请输入楼层号"
              className="user-form-input"
              style={{ width: 120 }}
              disabled={disabled}
            />
          </Form.Item>

          <Form.Item name="name" label="楼层名称">
            <AppInput
              placeholder="请输入楼层名称"
              className="user-form-input"
              style={{ width: 150 }}
              disabled={disabled}
            />
          </Form.Item>

          <Form.Item name="status" label="状态">
            {/* 形态分类器②④：状态枚举（业务禁搜），children → options */}
            <AppSelect
              placeholder="请选择状态"
              allowClear
              className="user-form-input"
              style={{ width: 120 }}
              disabled={disabled}
              showSearch={false}
              options={STATUS_OPTIONS}
            />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button
                type="primary"
                icon={<SearchOutlined />}
                onClick={onSearch}
                loading={loading}
                disabled={disabled}
              >
                搜索
              </Button>
              <Button onClick={onReset} disabled={disabled}>
                重置
              </Button>
              <Button icon={<ReloadOutlined />} onClick={onRefresh} loading={loading}>
                刷新
              </Button>
            </Space>
          </Form.Item>
        </Form>

        <Space>
          <Radio.Group
            value={viewMode}
            onChange={(e) => onViewModeChange(e.target.value)}
            buttonStyle="solid"
          >
            <Radio.Button value="table">
              <TableOutlined /> 表格
            </Radio.Button>
            <Radio.Button value="card">
              <AppstoreOutlined /> 卡片
            </Radio.Button>
          </Radio.Group>
          <Button icon={<ImportOutlined />} onClick={onImport}>
            导入
          </Button>
          <Button icon={<ExportOutlined />} onClick={onExport}>
            导出
          </Button>
          {selectedCount > 0 && (
            <Button
              icon={<DeleteOutlined />}
              onClick={onBatchDelete}
              style={{ color: "var(--theme-error)" }}
            >
              批量删除 ({selectedCount})
            </Button>
          )}
          <Button type="primary" icon={<PlusOutlined />} onClick={onAdd}>
            新增楼层
          </Button>
        </Space>
      </div>
    </Card>
  );
};
