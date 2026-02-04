/**
 * 楼层编辑弹窗组件
 */

import type { FC } from 'react';
import { Modal, Form, Input, InputNumber, Select } from 'antd';
import type { FormInstance } from 'antd/es/form';
import type { Floor } from '@/types';
import type { BuildingOption } from '../useBuildingOptions';
import { STATUS_OPTIONS, DEFAULT_FORM_VALUES } from '../constants';

const { Option } = Select;
const { TextArea } = Input;

interface FloorModalProps {
  visible: boolean;
  editingFloor: Floor | null;
  buildingOptions: BuildingOption[];
  form: FormInstance;
  onOk: () => void;
  onCancel: () => void;
}

export const FloorModal: FC<FloorModalProps> = ({
  visible,
  editingFloor,
  buildingOptions,
  form,
  onOk,
  onCancel,
}) => {
  const handleCancel = () => {
    form.resetFields();
    onCancel();
  };

  return (
    <Modal
      title={editingFloor ? '编辑楼层' : '新增楼层'}
      open={visible}
      onOk={onOk}
      onCancel={handleCancel}
      width={600}
    >
      <Form form={form} layout="horizontal" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
        <Form.Item
          name="buildingId"
          label="所属楼宇"
          rules={[{ required: true, message: '请选择所属楼宇' }]}
        >
          <Select placeholder="请选择所属楼宇" disabled={!!editingFloor}>
            {buildingOptions.map((b) => (
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
