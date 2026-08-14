/**
 * BatchExportModal 网络管理批量导出模态框组件
 * 支持选择多个实体类型一次性导出为 ZIP 文件
 */

import { useState, useMemo, type FC } from "react";
import { App, Modal, Checkbox, Space, Button } from "antd";
import { DownloadOutlined, CheckOutlined, CloseOutlined } from "@ant-design/icons";
import { getAccessToken } from "@/utils/authHelpers";

export interface EntityType {
  key: string;
  label: string;
}

export interface BatchExportModalProps {
  visible: boolean;
  onConfirm: (entityTypes: string[]) => Promise<void>;
  onCancel: () => void;
  loading?: boolean;
  availableEntityTypes?: EntityType[];
}

// 默认支持的 9 个实体类型
const DEFAULT_ENTITY_TYPES: EntityType[] = [
  { key: "devices", label: "网络设备" },
  { key: "credentials", label: "授权凭证" },
  { key: "templates", label: "配置模板" },
  { key: "commands", label: "命令分发" },
  { key: "executions", label: "配置执行" },
  { key: "backups", label: "配置备份" },
  { key: "discoveries", label: "设备发现" },
  { key: "mac", label: "MAC地址" },
  { key: "ports", label: "端口采集" },
];

const BatchExportModal: FC<BatchExportModalProps> = ({
  visible,
  onConfirm,
  onCancel,
  loading = false,
  availableEntityTypes = DEFAULT_ENTITY_TYPES,
}) => {
  // 默认全选所有实体类型
  const { message } = App.useApp();
  const [selectedTypes, setSelectedTypes] = useState<string[]>(
    DEFAULT_ENTITY_TYPES.map((e) => e.key)
  );

  // 当 modal 打开时重置为全选
  const handleOpen = () => {
    setSelectedTypes(DEFAULT_ENTITY_TYPES.map((e) => e.key));
  };

  // 全选
  const handleSelectAll = () => {
    setSelectedTypes(availableEntityTypes.map((e) => e.key));
  };

  // 清空
  const handleClearAll = () => {
    setSelectedTypes([]);
  };

  // 切换单个选项
  const handleToggle = (key: string) => {
    setSelectedTypes((prev) =>
      prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key]
    );
  };

  // 确认导出
  const handleConfirm = async () => {
    if (selectedTypes.length === 0) {
      message.warning("请至少选择一个实体类型");
      return;
    }
    await onConfirm(selectedTypes);
  };

  // 计算是否全选
  const isAllSelected = useMemo(
    () => selectedTypes.length === availableEntityTypes.length,
    [selectedTypes, availableEntityTypes]
  );

  // 计算是否部分选中
  const isIndeterminate = useMemo(
    () => selectedTypes.length > 0 && selectedTypes.length < availableEntityTypes.length,
    [selectedTypes, availableEntityTypes]
  );

  // 响应式列数配置
  const checkboxColSpan = useMemo(() => {
    // 桌面 3 列，平板 2 列，移动 1 列
    if (typeof window !== "undefined") {
      if (window.innerWidth >= 992) return 8;
      if (window.innerWidth >= 576) return 12;
    }
    return 24;
  }, []);

  return (
    <Modal
      title="批量导出网络管理数据"
      open={visible}
      onOk={handleConfirm}
      onCancel={onCancel}
      width={600}
      okText={`确认导出 (${selectedTypes.length})`}
      okButtonProps={{
        icon: <DownloadOutlined />,
        loading,
        disabled: selectedTypes.length === 0,
      }}
      cancelButtonProps={{ disabled: loading }}
      afterOpenChange={(open) => open && handleOpen()}
      destroyOnHidden
    >
      <Space direction="vertical" style={{ width: "100%" }} size="middle">
        {/* 快捷操作按钮 */}
        <Space>
          <Button size="small" onClick={handleSelectAll} disabled={loading}>
            全选
          </Button>
          <Button size="small" onClick={handleClearAll} disabled={loading}>
            清空
          </Button>
        </Space>

        {/* 实体类型选择列表 */}
        <div>
          <Checkbox
            indeterminate={isIndeterminate}
            onChange={(e) => (e.target.checked ? handleSelectAll() : handleClearAll())}
            checked={isAllSelected}
            disabled={loading}
          >
            全选（{selectedTypes.length}/{availableEntityTypes.length}）
          </Checkbox>
        </div>

        <div style={{ display: "flex", flexWrap: "wrap", gap: "8px" }}>
          {availableEntityTypes.map((entity) => (
            <Checkbox
              key={entity.key}
              checked={selectedTypes.includes(entity.key)}
              onChange={() => handleToggle(entity.key)}
              disabled={loading}
              style={
                typeof window !== "undefined" && window.innerWidth < 576 ? { width: "100%" } : {}
              }
            >
              {entity.label}
            </Checkbox>
          ))}
        </div>
      </Space>
    </Modal>
  );
};

export default BatchExportModal;
