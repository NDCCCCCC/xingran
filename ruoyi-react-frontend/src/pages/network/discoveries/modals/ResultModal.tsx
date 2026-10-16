/**
 * Discovery Result Modal
 * 发现结果模态框
 */

import { Modal, Table, Space, Button } from "antd";
import type { ColumnsType } from "antd/es/table";
import { ImportOutlined } from "@ant-design/icons";
import type { DeviceDiscovery } from "@/types";

export interface ResultModalProps {
  open: boolean;
  currentDiscovery: DeviceDiscovery | null;
  discoveredDevices: Record<string, unknown>[];
  onImport: () => Promise<void>;
  onClose: () => void;
}

// 发现结果表格列
const resultColumns: ColumnsType<Record<string, unknown>> = [
  { title: "设备名称", dataIndex: "deviceName", key: "deviceName" },
  { title: "IP地址", dataIndex: "ipAddress", key: "ipAddress" },
  { title: "厂商", dataIndex: "vendor", key: "vendor" },
  { title: "型号", dataIndex: "model", key: "model" },
  { title: "设备类型", dataIndex: "deviceType", key: "deviceType" },
];

export function ResultModal({
  open,
  currentDiscovery,
  discoveredDevices,
  onImport,
  onClose,
}: ResultModalProps) {
  return (
    <Modal
      title={`发现结果 - ${currentDiscovery?.taskName || ""}`}
      open={open}
      onCancel={onClose}
      width={900}
      footer={[
        <Button key="close" onClick={onClose}>
          关闭
        </Button>,
        <Button
          key="import"
          type="primary"
          icon={<ImportOutlined />}
          onClick={onImport}
          disabled={discoveredDevices.length === 0}
        >
          导入设备 ({discoveredDevices.length})
        </Button>,
      ]}
    >
      <div style={{ marginBottom: 16 }}>
        <Space>
          <span>总IP数: {currentDiscovery?.totalIPs || 0}</span>
          <span>发现设备: {discoveredDevices.length}</span>
        </Space>
      </div>

      <Table
        columns={resultColumns}
        dataSource={discoveredDevices}
        rowKey={(record) => String(record.ipAddress || record.id || Math.random())}
        pagination={false}
        size="small"
        scroll={{ y: 400 }}
      />
    </Modal>
  );
}
