/**
 * Network Command Dispatch Modal
 * 网络命令分发模态框
 */

import { Modal, Form, Table } from "antd";
// Phase 71 (W4)：裸 Input/TextArea → 契约组件
import { AppInput, AppTextArea } from "@/components/ui";
import type { NetworkDevice } from "@/types";
import { deviceColumns } from "../columns";

export interface CommandDispatchModalProps {
  open: boolean;
  devices: NetworkDevice[];
  selectedRowKeys: string[];
  /** FIX-01: 设备列表请求中，内嵌表显示 loading 蒙层 */
  loading?: boolean;
  onOk: () => Promise<void>;
  onCancel: () => void;
  onSelectionChange: (keys: React.Key[]) => void;
}

export function CommandDispatchModal({
  open,
  devices,
  selectedRowKeys,
  loading = false,
  onOk,
  onCancel,
  onSelectionChange,
}: CommandDispatchModalProps) {
  const [form] = Form.useForm();

  return (
    <Modal
      title="快速命令分发"
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
      okButtonProps={{ disabled: selectedRowKeys.length === 0 }}
      destroyOnHidden
    >
      <Form form={form} layout="vertical">
        <Form.Item label="选择设备" required>
          <Table
            loading={loading}
            rowSelection={{
              selectedRowKeys,
              onChange: (keys) => onSelectionChange(keys),
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
          name="commandContent"
          label="命令内容"
          rules={[{ required: true, message: "请输入命令内容" }]}
          extra="每行一条命令，支持标准网络设备命令"
        >
          <AppTextArea
            rows={8}
            placeholder={`示例：
display version
display interface
display ip routing-table

!
show version
show running-config
!`}
          />
        </Form.Item>

        <Form.Item
          name="timeout"
          label="超时时间(秒)"
          initialValue={300}
          rules={[{ required: true }]}
        >
          <AppInput type="number" min={10} max={3600} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
