import type { FC } from 'react';
import { Form, Input, Select, Button, Space, Radio, Card } from 'antd';
import {
  SearchOutlined,
  ReloadOutlined,
  TableOutlined,
  AppstoreOutlined,
  ImportOutlined,
  ExportOutlined,
  PlusOutlined,
  DeleteOutlined,
  BuildOutlined,
} from '@ant-design/icons';
import type { FormInstance } from 'antd/es/form';
import type { ViewMode } from '../constants';
import { STATUS_OPTIONS } from '../constants';
import type { Building } from '@/types';
import { DepartmentTreeSelect } from '@/components/shared/DepartmentTreeSelect';
import type { TreeNode } from '@/components/shared/DepartmentTreeSelect';

const { Option } = Select;

interface FloorSearchFormProps {
  form: FormInstance;
  buildingOptions: Building[];
  viewMode: ViewMode;
  loading: boolean;
  departmentTreeData: TreeNode[];
  buildingOptionsByDept: Building[];
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
  onDepartmentChange: (deptId: string) => void;
  selectedCount: number;
}

export const FloorSearchForm: FC<FloorSearchFormProps> = ({
  form,
  buildingOptions,
  viewMode,
  loading,
  departmentTreeData,
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
  onDepartmentChange,
  selectedCount,
}) => {
  return (
    <Card style={{ marginBottom: 16 }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-start',
          flexWrap: 'wrap',
          gap: '16px',
        }}
      >
        <Form form={form} layout="inline" style={{ flex: 1, minWidth: 0 }} disabled={disabled}>
          <Form.Item label="所属机构">
            <Space size="small" style={{ display: 'flex', alignItems: 'center' }}>
              <BuildOutlined style={{ color: '#1890ff' }} />
              <DepartmentTreeSelect
                style={{ width: 180 }}
                placeholder="选择机构"
                treeData={departmentTreeData}
                value={selectedDeptId}
                onChange={onDepartmentChange}
                allowClear
                disabled={disabled}
              />
            </Space>
          </Form.Item>

          <Form.Item name="buildingId" label="所属楼宇">
            <Select
              placeholder="请选择楼宇"
              allowClear
              style={{ width: 150 }}
              showSearch
              optionFilterProp="children"
              disabled={disabled}
            >
              {buildingOptionsByDept.map((b) => (
                <Option key={b.id} value={b.id}>
                  {b.name}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item name="floorNo" label="楼层号">
            <Input placeholder="请输入楼层号" allowClear style={{ width: 120 }} disabled={disabled} />
          </Form.Item>

          <Form.Item name="name" label="楼层名称">
            <Input placeholder="请输入楼层名称" allowClear style={{ width: 150 }} disabled={disabled} />
          </Form.Item>

          <Form.Item name="status" label="状态">
            <Select placeholder="请选择状态" allowClear style={{ width: 120 }} disabled={disabled}>
              {STATUS_OPTIONS.map((opt) => (
                <Option key={opt.value} value={opt.value}>
                  {opt.label}
                </Option>
              ))}
            </Select>
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
              <Button onClick={onReset} disabled={disabled}>重置</Button>
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
            <Button danger icon={<DeleteOutlined />} onClick={onBatchDelete}>
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
