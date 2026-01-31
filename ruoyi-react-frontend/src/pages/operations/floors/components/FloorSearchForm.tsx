/**
 * 楼层搜索表单组件
 */

import type { FC } from 'react';
import { Form, Input, Select, Button, Space, Radio } from 'antd';
import {
  SearchOutlined,
  ReloadOutlined,
  TableOutlined,
  AppstoreOutlined,
  ImportOutlined,
  ExportOutlined,
  PlusOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import type { FormInstance } from 'antd/es/form';
import type { ViewMode } from '../constants';
import { STATUS_OPTIONS } from '../constants';
import type { BuildingOption } from '../useBuildingOptions';

const { Option } = Select;

interface FloorSearchFormProps {
  form: FormInstance;
  buildingOptions: BuildingOption[];
  viewMode: ViewMode;
  loading: boolean;
  onSearch: () => void;
  onReset: () => void;
  onRefresh: () => void;
  onViewModeChange: (mode: ViewMode) => void;
  onImport: () => void;
  onExport: () => void;
  onBatchDelete: () => void;
  onAdd: () => void;
  selectedCount: number;
}

export const FloorSearchForm: FC<FloorSearchFormProps> = ({
  form,
  buildingOptions,
  viewMode,
  loading,
  onSearch,
  onReset,
  onRefresh,
  onViewModeChange,
  onImport,
  onExport,
  onBatchDelete,
  onAdd,
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
          <Form.Item name="buildingCode" label="所属楼宇">
            <Select placeholder="请选择楼宇" allowClear style={{ width: 150 }}>
              {buildingOptions.map((b) => (
                <Option key={b.code} value={b.code}>
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
