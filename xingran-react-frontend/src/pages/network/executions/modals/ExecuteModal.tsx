/**
 * Config Execute Modal
 * 配置执行模态框
 */

import { Modal, Form, Table, Card } from "antd";
// Phase 71 (W4)：裸 Input/Select → 契约组件
import { AppInput, AppSelect } from "@/components/ui";
import type { NetworkDevice, ConfigTemplate } from "@/types";
import { deviceColumns } from "../columns";

export interface ConfigExecuteModalProps {
  open: boolean;
  devices: NetworkDevice[];
  templates: ConfigTemplate[];
  selectedTemplate: ConfigTemplate | null;
  selectedRowKeys: string[];
  /** FIX-01: 设备列表请求中，内嵌表显示 loading 蒙层 */
  loading?: boolean;
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
  loading = false,
  onOk,
  onCancel,
  onTemplateChange,
  onSelectedRowKeysChange,
}: ConfigExecuteModalProps) {
  const [form] = Form.useForm();

  return (
    <Modal
      title="执行配置模板"
      mask={{ closable: false }}
      keyboard={false}
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
          <AppInput placeholder="请输入任务名称" />
        </Form.Item>

        <Form.Item
          name="templateId"
          label="选择模板"
          rules={[{ required: true, message: "请选择模板" }]}
        >
          {/* 形态分类器①/②：children 写法改写 options；原 showSearch + optionFilterProp="children"
              改为 options 形态后由 AppSelect 契约的 optionFilterProp="label" 承接（均带 label），
              模板可能较多 → 保留显式 showSearch */}
          <AppSelect
            placeholder="请选择配置模板"
            showSearch
            onChange={(value) => onTemplateChange(value as string)}
            onSearch={() => {}}
            options={templates.map((t) => ({
              value: t.id,
              label: `${t.templateName} (${t.templateCode})`,
            }))}
          />
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
                background: "var(--theme-bg-secondary)",
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
            loading={loading}
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
          <div style={{ marginTop: 8, color: "var(--theme-text-tertiary)" }}>
            已选择 {selectedRowKeys.length} 台设备
          </div>
        </Form.Item>

        <Form.Item
          name="timeout"
          label="超时时间(秒)"
          initialValue={300}
          rules={[{ required: true }]}
        >
          <AppInput type="number" min={10} max={3600} style={{ width: 200 }} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
