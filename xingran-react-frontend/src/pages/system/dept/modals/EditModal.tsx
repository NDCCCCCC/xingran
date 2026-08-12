/**
 * Department Edit Modal
 * 部门编辑模态框
 */

import { useEffect } from "react";
import { Modal, Form, Input, InputNumber, Select, Space, TreeSelect } from "antd";
import type { FormInstance } from "antd/es/form";
import { UserOutlined } from "@ant-design/icons";
import type { DeptUser, ParentOption, Department } from "../types";
import { STATUS_OPTIONS, EXTERNAL_ORG_OPTIONS } from "../constants";

const { Option } = Select;

export interface DeptEditModalProps {
  open: boolean;
  editingDept: Department | null;
  parentOptions: ParentOption[];
  deptUsers: DeptUser[];
  loadingUsers: boolean;
  form: FormInstance<unknown>;
  onOk: () => Promise<void>;
  onCancel: () => void;
  onParentChange: (value: string) => void;
  onLeaderChange: (userId: string) => void;
}

export function DeptEditModal({
  open,
  editingDept,
  parentOptions,
  deptUsers,
  loadingUsers,
  form,
  onOk,
  onCancel,
  onParentChange,
  onLeaderChange,
}: DeptEditModalProps) {
  // 当editingDept变化时，设置表单值
  useEffect(() => {
    if (open && editingDept) {
      form.setFieldsValue(editingDept);
    } else if (open && !editingDept) {
      // 新增模式，重置表单为默认值
      form.resetFields();
    }
  }, [open, editingDept, form]);

  return (
    <Modal
      title={editingDept ? "编辑部门" : "新增部门"}
      open={open}
      onOk={async () => {
        await onOk();
      }}
      onCancel={() => {
        onCancel();
        form.resetFields();
      }}
      width={600}
      destroyOnHidden
    >
      <Form form={form} layout="vertical">
        <Form.Item name="parentId" label="上级部门" initialValue="">
          <TreeSelect
            style={{ width: "100%" }}
            placeholder="请选择上级部门"
            treeData={[{ title: "顶级部门", value: "", key: "" }, ...parentOptions]}
            allowClear
            showSearch
            treeLine={{ showLeafIcon: false }}
            onChange={onParentChange}
          />
        </Form.Item>
        <Form.Item
          name="deptName"
          label="部门名称"
          rules={[{ required: true, message: "请输入部门名称" }]}
        >
          <Input />
        </Form.Item>
        <Form.Item
          name="deptCode"
          label="部门编码"
          rules={[{ required: true, message: "请输入部门编码" }]}
          tooltip="部门编码用于唯一标识部门，在Excel导入时作为关联键"
        >
          <Input placeholder="请输入部门编码，如：DEV、TECH" />
        </Form.Item>
        <Form.Item
          name="orderNum"
          label="显示顺序"
          rules={[{ required: true, message: "请输入显示顺序" }]}
          initialValue={0}
        >
          <InputNumber min={0} style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item name="leader" label="负责人">
          <Select
            placeholder="请选择负责人"
            loading={loadingUsers}
            showSearch
            allowClear
            optionFilterProp="label"
            onChange={onLeaderChange}
            notFoundContent={loadingUsers ? "加载中..." : "请先选择上级部门"}
           onSearch={() => {}}>
            {deptUsers.map((user) => (
              <Option
                key={user.id}
                value={user.id}
                label={`${user.nickname || user.username} (${user.username})`}
              >
                <Space>
                  <UserOutlined />
                  {user.nickname || user.username}
                  <span style={{ color: "var(--theme-text-tertiary, #999)" }}>({user.username})</span>
                </Space>
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="phone" hidden>
          <Input />
        </Form.Item>
        <Form.Item name="email" hidden>
          <Input />
        </Form.Item>
        <Form.Item
          name="isExternalOrg"
          label="外部机构"
          initialValue={0}
          tooltip="标记为外部机构的部门可以作为楼宇的所属机构"
        >
          <Select onSearch={() => {}}>
            {EXTERNAL_ORG_OPTIONS.map((opt) => (
              <Option key={opt.value} value={opt.value}>{opt.label}</Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="status" label="状态" initialValue={0}>
          <Select onSearch={() => {}}>
            {STATUS_OPTIONS.map((opt) => (
              <Option key={opt.value} value={opt.value}>{opt.label}</Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="remark" label="备注">
          <Input.TextArea rows={4} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
