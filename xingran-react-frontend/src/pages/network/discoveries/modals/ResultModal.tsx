/**
 * Discovery Result Modal
 * 发现结果模态框
 */

import { Modal, Table, Space, Button } from "antd";
import { ImportOutlined } from "@ant-design/icons";
import type { DeviceDiscovery } from "@/types";
import type { DictItem } from "@/hooks/useDict";
import { getDeviceColumns } from "../columns";

export interface ResultModalProps {
  open: boolean;
  currentDiscovery: DeviceDiscovery | null;
  discoveredDevices: Record<string, unknown>[];
  /** FIX-01: 结果请求中，内嵌表显示 loading 蒙层 */
  loading?: boolean;
  onImport: () => Promise<void>;
  /** Phase 86 W3 补漏：三态门控（`networkDiscovery.add` 就近归码）。
   *  ready 无权限 → 不渲染；未就绪 → 渲染但 disabled + hint；有权限 → 可用。 */
  importVisible?: boolean;
  importDisabled?: boolean;
  importHint?: string;
  /**
   * 设备类型字典（network_device_type）——由页面层 useDict 提供。
   * 不传 = 退化为静态映射（字典空态 / 接口异常），行为与迁移前一致。
   */
  deviceTypeDict?: DictItem[];
  onClose: () => void;
}

export function ResultModal({
  open,
  currentDiscovery,
  discoveredDevices,
  loading = false,
  onImport,
  importVisible = true,
  importDisabled = false,
  importHint,
  deviceTypeDict,
  onClose,
}: ResultModalProps) {
  // 列定义复用 discoveries/columns 工厂：字典经页面层透传，三级回退口径统一
  const resultColumns = getDeviceColumns<Record<string, unknown>>(deviceTypeDict);
  return (
    <Modal
      title={`发现结果 - ${currentDiscovery?.taskName || ""}`}
      mask={{ closable: false }}
      keyboard={false}
      open={open}
      onCancel={onClose}
      width={900}
      footer={[
        <Button key="close" onClick={onClose}>
          关闭
        </Button>,
        // Phase 86 W3 补漏：写入口三态门控（ready 无权限 → 不渲染；未就绪 → disabled + hint）
        importVisible && (
          <Button
            aria-label="导入结果"
            title={importDisabled ? (importHint ?? "导入结果") : "导入结果"}
            key="import"
            type="primary"
            icon={<ImportOutlined />}
            onClick={onImport}
            disabled={importDisabled || discoveredDevices.length === 0}
          >
            导入设备 ({discoveredDevices.length})
          </Button>
        ),
      ]}
    >
      <div style={{ marginBottom: 16 }}>
        <Space>
          <span>总IP数: {currentDiscovery?.totalIPs || 0}</span>
          <span>发现设备: {discoveredDevices.length}</span>
        </Space>
      </div>

      <Table
        loading={loading}
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
