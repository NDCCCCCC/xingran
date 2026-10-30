/**
 * Department Edit Modal
 * 部门编辑模态框
 */

import { useEffect, useState } from "react";
import { Modal, Form, InputNumber, Space, TreeSelect } from "antd";
// 负责人下拉仍用 children <Option> 形态（带 label + 复合子节点），故保留 Option 解构
import { Select } from "antd";
const { Option } = Select;
// Phase 71 (W2)：裸 Input/Select/Input.TextArea → 契约组件。
// 负责人下拉为「children + label + 复合子节点」形态：原已显式 optionFilterProp="label"，
// 且每个 <Option> 都带 label → 与 AppSelect 默认值同构，保留 children 写法（AppSelect 透传 children）。
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";
import type { FormInstance } from "antd/es/form";
import { UserOutlined } from "@ant-design/icons";
import type { DeptUser, ParentOption, Department } from "../types";
import { STATUS_OPTIONS, EXTERNAL_ORG_OPTIONS } from "../constants";

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
  const [submitting, setSubmitting] = useState(false);
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
        setSubmitting(true);
        try {
          await onOk();
        } finally {
          setSubmitting(false);
        }
      }}
      cancelButtonProps={{ disabled: submitting }}
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
          <AppInput />
        </Form.Item>
        <Form.Item
          name="deptCode"
          label="部门编码"
          rules={[{ required: true, message: "请输入部门编码" }]}
          tooltip="部门编码用于唯一标识部门，在Excel导入时作为关联键"
        >
          <AppInput placeholder="请输入部门编码，如：DEV、TECH" />
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
          <AppSelect
            placeholder="请选择负责人"
            loading={loadingUsers}
            showSearch
            allowClear
            optionFilterProp="label"
            // AppSelect 泛型被 ComponentProps 擦除 → onChange 值为 unknown，收敛回业务类型
            onChange={(value) => onLeaderChange(value as string)}
            notFoundContent={loadingUsers ? "加载中..." : "请先选择上级部门"}
            onSearch={() => {}}
          >
            {deptUsers.map((user) => (
              <Option
                key={user.id}
                value={user.id}
                label={`${user.nickname || user.username} (${user.username})`}
              >
                <Space>
                  <UserOutlined />
                  {user.nickname || user.username}
                  <span style={{ color: "var(--theme-text-tertiary)" }}>
                    ({user.username})
                  </span>
                </Space>
              </Option>
            ))}
          </AppSelect>
        </Form.Item>
        <Form.Item name="phone" hidden>
          <AppInput />
        </Form.Item>
        <Form.Item name="email" hidden>
          <AppInput />
        </Form.Item>
        <Form.Item
          name="isExternalOrg"
          label="外部机构"
          initialValue={0}
          tooltip="标记为外部机构的部门可以作为楼宇的所属机构"
        >
          {/* 形态分类器②④：外部机构固定枚举（业务禁搜） */}
          <AppSelect
            allowClear
            onSearch={() => {}}
            showSearch={false}
            // as const 只读元组 → antd options 需可变数组
            options={[...EXTERNAL_ORG_OPTIONS]}
          />
        </Form.Item>
        <Form.Item name="status" label="状态" initialValue={0}>
          {/* 形态分类器②④：状态枚举（业务禁搜） */}
          <AppSelect allowClear onSearch={() => {}} showSearch={false} options={[...STATUS_OPTIONS]} />
        </Form.Item>
        <Form.Item name="remark" label="备注">
          <AppTextArea rows={4} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
