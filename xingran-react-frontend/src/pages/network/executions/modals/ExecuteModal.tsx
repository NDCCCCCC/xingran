/**
 * Config Execute Modal
 * 配置执行模态框
 */

import { Modal, Form, Input, Select, Table, Card } from "antd";
import type { NetworkDevice, ConfigTemplate } from "@/types";
import { deviceColumns } from "../columns";

const { Option } = Select;

export interface ConfigExecuteModalProps {
  open: boolean;
  devices: NetworkDevice[];
  templates: ConfigTemplate[];
  selectedTemplate: ConfigTemplate | null;
  selectedRowKeys: string[];
  onOk: () => Promise<void>;
  onCancel: () => void;
  onTemplateChange: (templateId: string) => void;
  onSelectedRowKeysChange: (keys: string[]) => void;
}

export function ConfigExecuteModal({
  open,
  devices,
  templates,
  selectedTemplate,
  selectedRowKeys,
  onOk,
  onCancel,
  onTemplateChange,
  onSelectedRowKeysChange,
}: ConfigExecuteModalProps) {
  const [form] = Form.useForm();

  return (
    <Modal
      title="执行配置模板"
      open={open}
      onOk={async () => {
        await onOk();
        form.resetFields();
      }}
      onCancel={() => {
        onCancel();
        form.resetFields();
      }}
      width={900}
      okButtonProps={{ disabled: selectedRowKeys.length === 0 || !selectedTemplate }}
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="executionName"
          label="任务名称"
          rules={[{ required: true, message: "请输入任务名称" }]}
        >
          <Input placeholder="请输入任务名称" />
        </Form.Item>

        <Form.Item
          name="templateId"
          label="选择模板"
          rules={[{ required: true, message: "请选择模板" }]}
        >
          <Select
            placeholder="请选择配置模板"
            showSearch
            optionFilterProp="children"
            onChange={onTemplateChange}
            onSearch={() => {}}
          >
            {templates.map((t) => (
              <Option key={t.id} value={t.id}>
                {t.templateName} ({t.templateCode})
              </Option>
            ))}
          </Select>
        </Form.Item>

        {selectedTemplate && (
          <Card size="small" style={{ marginBottom: 16 }}>
            <div>
              <strong>模板预览:</strong>
            </div>
            <pre
              style={{
                maxHeight: 200,
                overflow: "auto",
                background: "#fafafa",
                padding: 12,
                marginTop: 8,
              }}
            >
              {selectedTemplate.templateContent}
            </pre>
          </Card>
        )}

        <Form.Item label="选择目标设备" required>
          <Table
            rowSelection={{
              selectedRowKeys,
              onChange: (keys) => onSelectedRowKeysChange(keys as string[]),
              type: "checkbox",
            }}
            columns={deviceColumns}
            dataSource={devices}
            rowKey="id"
            pagination={false}
            size="small"
            scroll={{ y: 200 }}
          />
          <div style={{ marginTop: 8, color: "var(--theme-text-tertiary, #999)" }}>
            已选择 {selectedRowKeys.length} 台设备
          </div>
        </Form.Item>

        <Form.Item
          name="timeout"
          label="超时时间(秒)"
          initialValue={300}
          rules={[{ required: true }]}
        >
          <Input type="number" min={10} max={3600} style={{ width: 200 }} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
