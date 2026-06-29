/**
 * Template Edit Modal
 * 模板编辑模态框
 */

import { Modal, Form, Input, Select, Space } from "antd";
import type { ConfigTemplate } from "@/types";
import { VENDOR_OPTIONS, DEVICE_TYPE_OPTIONS, TEMPLATE_TYPE_OPTIONS } from "../constants";

const { Option } = Select;
const { TextArea } = Input;

export interface TemplateEditModalProps {
  open: boolean;
  editingTemplate: ConfigTemplate | null;
  onOk: () => Promise<void>;
  onCancel: () => void;
}

export function TemplateEditModal({
  open,
  editingTemplate,
  onOk,
  onCancel,
}: TemplateEditModalProps) {
  const [form] = Form.useForm();

  return (
    <Modal
      title={editingTemplate ? "编辑配置模板" : "新增配置模板"}
      open={open}
      onOk={async () => {
        await onOk();
      }}
      onCancel={() => {
        onCancel();
        form.resetFields();
      }}
      width={900}
      destroyOnHidden
    >
      <Form form={form} labelCol={{ span: 4 }} wrapperCol={{ span: 18 }}>
        <Form.Item
          name="templateName"
          label="模板名称"
          rules={[{ required: true, message: "请输入模板名称" }]}
        >
          <Input placeholder="请输入模板名称" />
        </Form.Item>
        <Form.Item
          name="templateCode"
          label="模板编码"
          rules={[{ required: true, message: "请输入模板编码" }]}
        >
          <Input placeholder="请输入模板编码（英文，用于API调用）" disabled={!!editingTemplate} />
        </Form.Item>
        <Form.Item name="templateType" label="模板类型" rules={[{ required: true }]}>
          <Select onSearch={() => {}}>
            {TEMPLATE_TYPE_OPTIONS.map((opt) => (
              <Option key={opt.value} value={opt.value}>
                {opt.label}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Space size="large">
          <Form.Item name="vendor" label="适用厂商">
            <Select placeholder="通用" allowClear style={{ width: 150 }} onSearch={() => {}}>
              {VENDOR_OPTIONS.map((opt) => (
                <Option key={opt.value} value={opt.value}>
                  {opt.label}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="deviceType" label="适用设备">
            <Select placeholder="通用" allowClear style={{ width: 150 }} onSearch={() => {}}>
              {DEVICE_TYPE_OPTIONS.map((opt) => (
                <Option key={opt.value} value={opt.value}>
                  {opt.label}
                </Option>
              ))}
            </Select>
          </Form.Item>
        </Space>
        <Form.Item
          name="templateContent"
          label="模板内容"
          rules={[{ required: true, message: "请输入模板内容" }]}
          extra="使用 {{.变量名}} 语法定义变量"
        >
          <TextArea
            rows={15}
            placeholder={`示例配置模板：
interface {{.InterfaceName}}
 description {{.Description}}
 ip address {{.IPAddress}} {{.SubnetMask}}

!
vlan {{.VLANID}}
 name {{.VLANName}}
!
`}
          />
        </Form.Item>
        <Form.Item
          name="variables"
          label="变量定义"
          extra="JSON格式，定义变量的默认值和描述"
        >
          <TextArea
            rows={6}
            placeholder={`示例：
{
  "InterfaceName": {"default": "GigabitEthernet0/0/1", "description": "接口名称"},
  "Description": {"default": "Uplink", "description": "接口描述"},
  "IPAddress": {"default": "192.168.1.1", "description": "IP地址"},
  "SubnetMask": {"default": "255.255.255.0", "description": "子网掩码"}
}`}
          />
        </Form.Item>
        <Form.Item name="description" label="备注">
          <TextArea rows={2} placeholder="请输入备注" />
        </Form.Item>
      </Form>
    </Modal>
  );
}
