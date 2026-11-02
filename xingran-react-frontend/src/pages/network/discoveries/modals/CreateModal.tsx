/**
 * Create Discovery Modal
 * 创建发现任务模态框
 */

import { Modal, Form, InputNumber, Switch } from "antd";
// Phase 71 (W4)：裸 Input/Select/TextArea → 契约组件（Select children 形态改写 options）
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";
import type { FormInstance } from "antd/es/form";
import type { Department } from "@/types";
import { DISCOVERY_TYPE_OPTIONS } from "../constants";

export interface CreateModalProps {
  open: boolean;
  departments: Department[];
  onOk: (form: FormInstance<unknown>) => Promise<void>;
  onCancel: () => void;
}

export function CreateModal({ open, departments, onOk, onCancel }: CreateModalProps) {
  const [form] = Form.useForm();
  const discoveryType = Form.useWatch("discoveryType", form);

  return (
    <Modal
      title="创建设备发现任务"
      mask={{ closable: false }}
      keyboard={false}
      open={open}
      onOk={() => onOk(form)}
      onCancel={() => {
        onCancel();
        form.resetFields();
      }}
      width={700}
      destroyOnHidden
    >
      <Form form={form} labelCol={{ span: 6 }} wrapperCol={{ span: 16 }}>
        <Form.Item
          name="taskName"
          label="任务名称"
          rules={[{ required: true, message: "请输入任务名称" }]}
        >
          <AppInput placeholder="请输入任务名称" />
        </Form.Item>

        <Form.Item name="discoveryType" label="发现类型" rules={[{ required: true }]}>
          {/* 形态分类器①/②：children 写法改写 options；固定 2 项 → 显式禁搜 */}
          <AppSelect onSearch={() => {}} showSearch={false} options={[...DISCOVERY_TYPE_OPTIONS]} />
        </Form.Item>

        <Form.Item
          name="ipRanges"
          label="IP范围"
          rules={[{ required: true, message: "请输入IP范围" }]}
          extra="每行一个IP范围，支持CIDR格式"
        >
          <AppTextArea
            rows={4}
            placeholder={`示例：
192.168.1.0/24
192.168.2.1-192.168.2.100
10.0.0.1-10.0.0.50`}
          />
        </Form.Item>

        {discoveryType === "snmp" && (
          <>
            <Form.Item
              name="snmpCommunity"
              label="SNMP Community"
              rules={[{ required: true, message: "请输入SNMP Community" }]}
            >
              <AppInput placeholder="请输入SNMP Community，如：public" />
            </Form.Item>
            <Form.Item name="snmpPort" label="SNMP端口" rules={[{ required: true }]}>
              <InputNumber min={1} max={65535} style={{ width: "100%" }} />
            </Form.Item>
          </>
        )}

        <Form.Item name="groupId" label="导入部门">
          {/* 形态分类器①/②：children 写法改写 options。部门为 UUID 外键，原 allowClear=true
              行为保持不变（onChange(undefined) → 不指定导入部门，后端按可空处理） */}
          <AppSelect
            placeholder="请选择部门"
            allowClear
            onSearch={() => {}}
            options={departments.map((dept) => ({ value: dept.id, label: dept.deptName }))}
          />
        </Form.Item>

        <Form.Item
          name="autoImport"
          label="自动导入"
          valuePropName="checked"
          extra="自动导入发现的设备到设备列表"
        >
          <Switch />
        </Form.Item>
      </Form>
    </Modal>
  );
}
