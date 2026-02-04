/**
 * 楼层搜索表单组件
 */

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
  departmentTreeData: TreeNode[];  // 外部机构部门树数据
  buildingOptionsByDept: Building[];  // 当前部门的楼宇选项
  selectedDeptId: string;  // 当前选中的部门ID
  onSearch: () => void;
  onReset: () => void;
  onRefresh: () => void;
  onViewModeChange: (mode: ViewMode) => void;
  onImport: () => void;
  onExport: () => void;
  onBatchDelete: () => void;
  onAdd: () => void;
  onDepartmentChange: (deptId: string) => void;  // 部门切换回调
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
        <Form form={form} layout="inline" style={{ flex: 1, minWidth: 0 }}>
          {/* 部门选择器（外部机构） */}
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
              />
            </Space>
          </Form.Item>

          {/* 楼宇选择器（根据选择的部门过滤） */}
          <Form.Item name="buildingId" label="所属楼宇">
            <Select
              placeholder="请选择楼宇"
              allowClear
              style={{ width: 150 }}
              showSearch
              optionFilterProp="children"
            >
              {buildingOptionsByDept.map((b) => (
                <Option key={b.id} value={b.id}>
                  {b.name}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item name="floorNo" label="楼层号">
            <Input placeholder="请输入楼层号" allowClear style={{ width: 120 }} />
          </Form.Item>
          <Form.Item name="name" label="楼层名称">
            <Input placeholder="请输入楼层名称" allowClear style={{ width: 150 }} />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select placeholder="请选择状态" allowClear style={{ width: 120 }}>
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
              >
                搜索
              </Button>
              <Button onClick={onReset}>重置</Button>
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
