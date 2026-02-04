/**
 * 楼层编辑弹窗组件
 */

import type { FC } from 'react';
import { Modal, Form, Input, InputNumber, Select } from 'antd';
import type { FormInstance } from 'antd/es/form';
import type { Floor, Building } from '@/types';
import { STATUS_OPTIONS, DEFAULT_FORM_VALUES } from '../constants';
import { DepartmentTreeSelect } from '@/components/shared/DepartmentTreeSelect';
import type { TreeNode } from '@/components/shared/DepartmentTreeSelect';

const { Option } = Select;
const { TextArea } = Input;

interface FloorModalProps {
  visible: boolean;
  editingFloor: Floor | null;
  buildingOptions: Building[];  // 当前部门的楼宇选项（未使用，保留兼容性）
  departmentTreeData: TreeNode[];  // 外部机构部门树数据
  buildingOptionsByDept: Building[];  // 当前选中的部门的楼宇选项
  selectedDeptId: string;  // 当前选中的部门ID
  form: FormInstance;
  onOk: () => void;
  onCancel: () => void;
  onDepartmentChange: (deptId: string) => void;  // 部门切换回调
}

export const FloorModal: FC<FloorModalProps> = ({
  visible,
  editingFloor,
  buildingOptions,
  departmentTreeData,
  buildingOptionsByDept,
  selectedDeptId,
  form,
  onOk,
  onCancel,
  onDepartmentChange,
}) => {
  const handleCancel = () => {
    form.resetFields();
    onCancel();
  };

  // 编辑模式下，禁用部门和楼宇选择
  const isDisabled = !!editingFloor;

  return (
    <Modal
      title={editingFloor ? '编辑楼层' : '新增楼层'}
      open={visible}
      onOk={onOk}
      onCancel={handleCancel}
      width={600}
    >
      <Form form={form} layout="horizontal" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
        {/* 部门选择器（外部机构）- 仅用于过滤楼宇，不作为表单字段提交 */}
        <Form.Item label="所属机构" required>
          <DepartmentTreeSelect
            placeholder="选择机构"
            treeData={departmentTreeData}
            value={selectedDeptId}
            onChange={onDepartmentChange}
            allowClear={!isDisabled}
            disabled={isDisabled}
          />
        </Form.Item>

        {/* 楼宇选择器（根据选择的部门过滤） */}
        <Form.Item
          name="buildingId"
          label="所属楼宇"
          rules={[{ required: true, message: '请选择所属楼宇' }]}
        >
          <Select
            placeholder="请选择所属楼宇"
            disabled={isDisabled}
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

        <Form.Item
          name="floorNo"
          label="楼层号"
          rules={[{ required: true, message: '请输入楼层号' }]}
        >
          <Input placeholder="请输入楼层号，如：1F" disabled={!!editingFloor} />
        </Form.Item>
        <Form.Item name="name" label="楼层名称">
          <Input placeholder="请输入楼层名称" />
        </Form.Item>
        <Form.Item name="area" label="面积">
          <InputNumber min={0} placeholder="请输入面积(m²)" style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item
          name="status"
          label="状态"
          rules={[{ required: true, message: '请选择状态' }]}
          initialValue={DEFAULT_FORM_VALUES.status}
        >
          <Select placeholder="请选择状态">
            {STATUS_OPTIONS.map((opt) => (
              <Option key={opt.value} value={opt.value}>
                {opt.label}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="description" label="描述">
          <TextArea rows={3} placeholder="请输入描述" />
        </Form.Item>
      </Form>
    </Modal>
  );
};
