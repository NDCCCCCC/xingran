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
  /**
   * Phase 86 W2 通道①：导入触发回调 —— 由**必填放宽为可选**。
   * 不传 ⇒ 「导入」按钮**不渲染**（页面层用「ready 无权限 → 不传」表达「不在 DOM」）；
   * 既有调用方传了回调时行为零变更。
   */
  onImport?: () => void;
  /** Phase 86 W2 通道①：导出触发回调 —— 同上，由必填放宽为可选。 */
  onExport?: () => void;
  /** Phase 86 W2 通道①：导入按钮 disabled 位（未就绪 → 渲染但禁用）。 */
  importDisabled?: boolean;
  /** Phase 86 W2 通道①：导出按钮 disabled 位（未就绪 → 渲染但禁用）。 */
  exportDisabled?: boolean;
  /** 未就绪提示（透传到导入/导出按钮 title）。 */
  notReadyHint?: string;
  /** 批量删除回调（保持**必填** —— 页面层用 noop 包装兜底）。 */
  onBatchDelete: () => void;
  /** Phase 86 W2 通道②：批量删除按钮 disabled 位（未就绪 → 渲染但禁用）。 */
  batchDeleteDisabled?: boolean;
  /**
   * Phase 86 W2 通道②：批量删除按钮**渲不渲染**。
   * ready 无权限 → `false` → 按钮不在 DOM；未就绪 → `true` + `batchDeleteDisabled`
   * → 渲染但禁用（三态语义）。默认 `true` 保持既有调用方行为零变更。
   */
  showBatchDelete?: boolean;
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
  importDisabled = false,
  exportDisabled = false,
  notReadyHint,
  onBatchDelete,
  batchDeleteDisabled = false,
  showBatchDelete = true,
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
          {/* Phase 86 W2 通道①：导入/导出的门控点在**触发按钮**（不是 modal 容器）。
              `onImport` 不传 ⇒ 按钮不渲染（ready 无权限）；未就绪 ⇒ 渲染但 disabled + hint。 */}
          {onImport && (
            <Button
              icon={<ImportOutlined />}
              onClick={onImport}
              disabled={importDisabled}
              title={importDisabled ? notReadyHint : undefined}
            >
              导入
            </Button>
          )}
          {onExport && (
            <Button
              icon={<ExportOutlined />}
              onClick={onExport}
              disabled={exportDisabled}
              title={exportDisabled ? notReadyHint : undefined}
            >
              导出
            </Button>
          )}
          {/* Phase 86 W2 通道②：`showBatchDelete` 决定渲不渲染（ready 无权限 → 不在 DOM）；
              `batchDeleteDisabled` 决定禁不禁用（未就绪 → 渲染但 disabled + hint）。 */}
          {showBatchDelete && selectedCount > 0 && (
            <Button
              icon={<DeleteOutlined />}
              onClick={onBatchDelete}
              disabled={batchDeleteDisabled}
              title={batchDeleteDisabled ? notReadyHint : undefined}
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
