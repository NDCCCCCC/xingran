/**
 * Discovery Result Modal
 * 发现结果模态框
 */

import { Modal, Table, Space, Button } from 'antd';
import { ImportOutlined } from '@ant-design/icons';
import type { DeviceDiscovery, NetworkDevice } from '@/types';
import { deviceColumns } from '../columns';

export interface ResultModalProps {
  open: boolean;
  currentDiscovery: DeviceDiscovery | null;
  discoveredDevices: NetworkDevice[];
  onImport: () => Promise<void>;
  onClose: () => void;
}

export function ResultModal({
  open,
  currentDiscovery,
  discoveredDevices,
  onImport,
  onClose,
}: ResultModalProps) {
  return (
    <Modal
      title={`发现结果 - ${currentDiscovery?.taskName || ''}`}
      open={open}
      onCancel={onClose}
      width={900}
      footer={[
        <Button key="close" onClick={onClose}>关闭</Button>,
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
        columns={deviceColumns}
        dataSource={discoveredDevices}
        rowKey={(record) => record.ipAddress}
        pagination={false}
        size="small"
        scroll={{ y: 400 }}
      />
    </Modal>
  );
}
