/**
 * Create Discovery Modal
 * 创建发现任务模态框
 */

import { Modal, Form, Input, InputNumber, Select, Switch } from "antd";
import type { FormInstance } from "antd/es/form";
import type { Department } from "@/types";
import { DISCOVERY_TYPE_OPTIONS } from "../constants";

const { Option } = Select;
const { TextArea } = Input;

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
        <Form.Item name="taskName" label="任务名称" rules={[{ required: true, message: "请输入任务名称" }]}>
          <Input placeholder="请输入任务名称" />
        </Form.Item>

        <Form.Item name="discoveryType" label="发现类型" rules={[{ required: true }]}>
          <Select onSearch={() => {}}>
            {DISCOVERY_TYPE_OPTIONS.map(opt => (
              <Option key={opt.value} value={opt.value}>{opt.label}</Option>
            ))}
          </Select>
        </Form.Item>

        <Form.Item name="ipRanges" label="IP范围" rules={[{ required: true, message: "请输入IP范围" }]} extra="每行一个IP范围，支持CIDR格式">
          <TextArea
            rows={4}
            placeholder={`示例：
192.168.1.0/24
192.168.2.1-192.168.2.100
10.0.0.1-10.0.0.50`}
          />
        </Form.Item>

        {discoveryType === "snmp" && (
          <>
            <Form.Item name="snmpCommunity" label="SNMP Community" rules={[{ required: true, message: "请输入SNMP Community" }]}>
              <Input placeholder="请输入SNMP Community，如：public" />
            </Form.Item>
            <Form.Item name="snmpPort" label="SNMP端口" rules={[{ required: true }]}>
              <InputNumber min={1} max={65535} style={{ width: "100%" }} />
            </Form.Item>
          </>
        )}

        <Form.Item name="groupId" label="导入部门">
          <Select placeholder="请选择部门" allowClear onSearch={() => {}}>
            {departments.map(dept => (
              <Option key={dept.id} value={dept.id}>{dept.deptName}</Option>
            ))}
          </Select>
        </Form.Item>

        <Form.Item name="autoImport" label="自动导入" valuePropName="checked" extra="自动导入发现的设备到设备列表">
          <Switch />
        </Form.Item>
      </Form>
    </Modal>
  );
}
