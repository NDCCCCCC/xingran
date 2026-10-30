/**
 * Template Edit Modal
 * 模板编辑模态框
 */

import { Modal, Form, Space } from "antd";
// Phase 71 (W4)：裸 Input/Select/TextArea → 契约组件
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";
import type { ConfigTemplate } from "@/types";
import { VENDOR_OPTIONS, DEVICE_TYPE_OPTIONS, TEMPLATE_TYPE_OPTIONS } from "../constants";

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
      mask={{ closable: false }}
      keyboard={false}
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
          <AppInput placeholder="请输入模板名称" />
        </Form.Item>
        <Form.Item
          name="templateCode"
          label="模板编码"
          rules={[{ required: true, message: "请输入模板编码" }]}
        >
          <AppInput
            placeholder="请输入模板编码（英文，用于API调用）"
            disabled={!!editingTemplate}
          />
        </Form.Item>
        <Form.Item name="templateType" label="模板类型" rules={[{ required: true }]}>
          {/* 形态分类器①/②：children 写法改写 options；固定 3 项 → 显式禁搜 */}
          <AppSelect onSearch={() => {}} showSearch={false} options={TEMPLATE_TYPE_OPTIONS} />
        </Form.Item>
        <Space size="large">
          <Form.Item name="vendor" label="适用厂商">
            {/* 形态分类器①/②：children 改写 options；厂商固定 5 项 → 显式禁搜 */}
            <AppSelect
              placeholder="通用"
              style={{ width: 150 }}
              onSearch={() => {}}
              showSearch={false}
              options={VENDOR_OPTIONS}
            />
          </Form.Item>
          <Form.Item name="deviceType" label="适用设备">
            {/* 形态分类器①/②：children 改写 options；设备类型固定 5 项 → 显式禁搜 */}
            <AppSelect
              placeholder="通用"
              style={{ width: 150 }}
              onSearch={() => {}}
              showSearch={false}
              options={DEVICE_TYPE_OPTIONS}
            />
          </Form.Item>
        </Space>
        <Form.Item
          name="templateContent"
          label="模板内容"
          rules={[{ required: true, message: "请输入模板内容" }]}
          extra="使用 {{.变量名}} 语法定义变量"
        >
          <AppTextArea
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
        <Form.Item name="variables" label="变量定义" extra="JSON格式，定义变量的默认值和描述">
          <AppTextArea
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
          <AppTextArea rows={2} placeholder="请输入备注" />
        </Form.Item>
      </Form>
    </Modal>
  );
}
