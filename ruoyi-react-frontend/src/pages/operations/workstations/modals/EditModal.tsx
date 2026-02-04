/**
 * Workstation Edit Modal
 * 工位编辑模态框
 */

import { Modal, Form, Input, InputNumber, Select, Row, Col } from 'antd';
import type { WorkstationOps } from '@/types';
import type { DepartmentTreeNode } from '@/types';
import { DepartmentTreeSelect } from '@/components/shared';
import { STATUS_OPTIONS, TYPE_OPTIONS } from '../constants';

const { Option } = Select;
const { TextArea } = Input;

export interface WorkstationEditModalProps {
  open: boolean;
  editingWorkstation: WorkstationOps | null;
  floorOptions: { id: string; code: string; name: string }[];
  deptTreeData: DepartmentTreeNode[];
  userOptions: { id: string; name: string }[];
  onOk: () => Promise<void>;
  onCancel: () => void;
  onDeptChange: (deptId: string) => void;
}

export function WorkstationEditModal({
  open,
  editingWorkstation,
  floorOptions,
  deptTreeData,
  userOptions,
  onOk,
  onCancel,
  onDeptChange,
}: WorkstationEditModalProps) {
  const [form] = Form.useForm();

  return (
    <Modal
      title={editingWorkstation ? '编辑工位' : '新增工位'}
      open={open}
      onOk={async () => {
        await onOk();
      }}
      onCancel={() => {
        onCancel();
        form.resetFields();
      }}
      width={700}
      destroyOnHidden
    >
      <Form form={form} layout="horizontal" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
        <Form.Item name="floorId" label="所属楼层" rules={[{ required: true, message: '请选择所属楼层' }]}>
          <Select placeholder="请选择所属楼层">
            {floorOptions.map((f) => (
              <Option key={f.id} value={f.id}>
                {f.name}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="name" label="工位名称" rules={[{ required: true, message: '请输入工位名称' }]}>
          <Input placeholder="请输入工位名称" />
        </Form.Item>
        <Form.Item name="type" label="工位类型" rules={[{ required: true, message: '请选择工位类型' }]}>
          <Select placeholder="请选择工位类型">
            {TYPE_OPTIONS.map((opt) => (
              <Option key={opt.value} value={opt.value}>
                {opt.label}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="deptId" label="所属部门">
              <DepartmentTreeSelect
                treeData={deptTreeData}
                onChange={() => {
                  // 当部门变化时，清空用户选择并重新加载用户
                  form.setFieldsValue({ userId: undefined });
                  const deptId = form.getFieldValue('deptId');
                  onDeptChange(deptId);
                }}
              />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="userId" label="所属用户">
              <Select placeholder="请先选择部门" allowClear>
                {userOptions.map((u) => (
                  <Option key={u.id} value={u.id}>
                    {u.name}
                  </Option>
                ))}
              </Select>
            </Form.Item>
          </Col>
        </Row>
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="positionX" label="X坐标">
              <InputNumber placeholder="X坐标" style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="positionY" label="Y坐标">
              <InputNumber placeholder="Y坐标" style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>
        <Form.Item name="status" label="状态" rules={[{ required: true, message: '请选择状态' }]}>
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
}
