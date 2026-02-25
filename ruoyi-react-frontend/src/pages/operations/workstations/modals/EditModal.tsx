/**
 * Workstation Edit Modal
 * 工位编辑模态框
 */

import { useLayoutEffect } from 'react';
import { Modal, Form, Input, InputNumber, Select, Row, Col, Cascader } from 'antd';
import type { FormInstance } from 'antd';
import type { WorkstationOps } from '@/types';
import type { DeptTreeNode } from '../types';
import { DepartmentTreeSelect } from '@/components/shared';
import { STATUS_OPTIONS, TYPE_OPTIONS } from '../constants';

const { Option } = Select;
const { TextArea } = Input;

// Cascader 选项类型
type CascaderOption = {
  value: string;
  label: string;
  children?: CascaderOption[];
  isLeaf?: boolean;
};

export interface WorkstationEditModalProps {
  open: boolean;
  form: FormInstance;
  editingWorkstation: WorkstationOps | null;
  deptTreeData: DeptTreeNode[];
  userOptions: { id: string; name: string }[];
  cascaderOptions: CascaderOption[];
  loadingCascader: boolean;
  handleCascaderLoadData: (selectedOptions: CascaderOption[]) => Promise<void> | void;
  onOk: (values: Record<string, unknown>) => Promise<void>;
  onCancel: () => void;
  onDeptChange: (deptId: string) => void;
  onOrgChange: (orgId: string) => void;
}

export function WorkstationEditModal({
  open,
  form,
  editingWorkstation,
  deptTreeData,
  userOptions,
  cascaderOptions,
  loadingCascader,
  handleCascaderLoadData,
  onOk,
  onCancel,
  onDeptChange,
  onOrgChange,
}: WorkstationEditModalProps) {
  // 仅在新增模式下重置表单（编辑模式的值由父组件设置）
  useLayoutEffect(() => {
    if (open && !editingWorkstation?.id) {
      form.resetFields();
      form.setFieldsValue({ status: 0, type: 0 });
    }
  }, [open, editingWorkstation, form]);

  return (
    <Modal
      title={editingWorkstation ? '编辑工位' : '新增工位'}
      open={open}
      onOk={async () => {
        try {
          // 验证表单并获取值
          const values = await form.validateFields();
          // 将表单值传递给父组件的 onOk 处理
          await onOk(values);
        } catch (error) {
          // 表单验证失败，不关闭模态框
          throw error;
        }
      }}
      onCancel={() => {
        onCancel();
        form.resetFields();
      }}
      width={700}
      destroyOnHidden
    >
      <Form form={form} layout="horizontal" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
        <Form.Item name="orgId" label="所属机构" rules={[{ required: true, message: '请选择所属机构' }]}>
          <DepartmentTreeSelect
            treeData={deptTreeData}
            placeholder="请选择机构"
            onChange={(value) => {
              // 当机构变化时，清空楼宇-楼层选择并重新加载楼宇
              form.setFieldsValue({ floorId: undefined });
              onOrgChange(value);
            }}
          />
        </Form.Item>
        <Form.Item name="floorId" label="所属楼层" rules={[{ required: true, message: '请选择所属楼层' }]}>
          <Cascader
            options={cascaderOptions}
            loadData={handleCascaderLoadData}
            loading={loadingCascader}
            placeholder="请先选择楼宇，再选择楼层"
            changeOnSelect
            showSearch={{ filter: (inputValue, path) => path.some(option => option.label?.toLowerCase().includes(inputValue.toLowerCase())) }}
          />
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
                placeholder="请选择部门"
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
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="width" label="宽度" initialValue={160}>
              <InputNumber placeholder="宽度" min={50} max={500} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="depth" label="深度" initialValue={70}>
              <InputNumber placeholder="深度" min={30} max={300} style={{ width: '100%' }} />
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
