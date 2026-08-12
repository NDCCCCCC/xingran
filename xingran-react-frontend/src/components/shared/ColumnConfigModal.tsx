import React, { useState, useMemo } from "react";
import {
  Modal,
  Checkbox,
  Input,
  InputNumber,
  Button,
  Space,
} from "antd";
import { SearchOutlined, HolderOutlined } from "@ant-design/icons";
import { DndContext, closestCenter } from "@dnd-kit/core";
import type { DragEndEvent } from "@dnd-kit/core";
import { SortableContext, useSortable, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import type { ColumnConfig } from "@/hooks/useColumnConfig";

interface SortableColumnItemProps {
  config: ColumnConfig;
  onToggleVisible: (visible: boolean) => void;
  onWidthChange: (width: number | null) => void;
}

// 可排序的列项组件
const SortableColumnItem: React.FC<SortableColumnItemProps> = ({
  config,
  onToggleVisible,
  onWidthChange,
}) => {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: config.key });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="flex items-center gap-3 p-3 bg-white border border-gray-200 rounded hover:bg-gray-50"
    >
      {/* 拖拽手柄 */}
      <div
        {...attributes}
        {...listeners}
        className="cursor-move text-gray-400 hover:text-gray-600"
      >
        <HolderOutlined />
      </div>

      {/* 复选框 */}
      <Checkbox
        checked={config.visible}
        onChange={(e) => onToggleVisible(e.target.checked)}
      >
        <span className="ml-2">{config.label}</span>
      </Checkbox>

      {/* 宽度输入 */}
      <InputNumber
        value={config.width || 0}
        onChange={onWidthChange}
        placeholder="宽度"
        min={50}
        max={500}
        step={10}
        className="w-24"
        size="small"
      />

      <span className="text-gray-400 text-sm">px</span>
    </div>
  );
};

export interface ColumnConfigModalProps {
  visible: boolean;
  config: ColumnConfig[];
  defaultConfig: ColumnConfig[];
  onSave: (config: ColumnConfig[]) => void;
  onReset: () => void;
  onClose: () => void;
  saving?: boolean;
}

export const ColumnConfigModal: React.FC<ColumnConfigModalProps> = ({
  visible,
  config,
  defaultConfig,
  onSave,
  onReset,
  onClose,
  saving = false,
}) => {
  const [searchValue, setSearchValue] = useState("");
  const [localConfig, setLocalConfig] = useState<ColumnConfig[]>(config);

  // 当弹窗打开时重置本地配置
  React.useEffect(() => {
    if (visible) {
      setLocalConfig(config);
    }
  }, [visible, config]);

  // 过滤配置
  const filteredConfig = useMemo(() => {
    if (!searchValue) return localConfig;
    const lowerSearch = searchValue.toLowerCase();
    return localConfig.filter(col =>
      col.label.toLowerCase().includes(lowerSearch) ||
      col.key.toLowerCase().includes(lowerSearch)
    );
  }, [localConfig, searchValue]);

  // 全选/取消全选
  const allChecked = localConfig.length > 0 && localConfig.every(c => c.visible);
  const someChecked = localConfig.some(c => c.visible) && !allChecked;

  const handleToggleAll = (checked: boolean) => {
    setLocalConfig(prev => prev.map(col => ({ ...col, visible: checked })));
  };

  // 拖拽结束
  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (over && active.id !== over.id) {
      setLocalConfig(prev => {
        const oldIndex = prev.findIndex(col => col.key === active.id);
        const newIndex = prev.findIndex(col => col.key === over.id);
        const newConfig = [...prev];
        const [removed] = newConfig.splice(oldIndex, 1);
        newConfig.splice(newIndex, 0, removed);
        return newConfig.map((col, idx) => ({ ...col, order: idx + 1 }));
      });
    }
  };

  // 处理保存
  const handleSave = async () => {
    await onSave(localConfig);
    // 保存成功后关闭弹窗
    onClose();
  };

  // 处理重置
  const handleReset = () => {
    Modal.confirm({
      title: "确认重置",
      content: "确定要重置列配置吗？此操作将清除您的个人设置。",
      okText: "确定",
      cancelText: "取消",
      onOk: () => {
        onReset();
        onClose();
      },
    });
  };

  return (
    <Modal
      title="列配置"
      open={visible}
      onCancel={onClose}
      width={700}
      footer={[
        <Button key="reset" onClick={handleReset}>
          重置
        </Button>,
        <Button key="cancel" onClick={onClose}>
          取消
        </Button>,
        <Button
          key="save"
          type="primary"
          onClick={handleSave}
          loading={saving}
        >
          确定
        </Button>,
      ]}
    >
      {/* 搜索框 */}
      <Input
        placeholder="搜索列..."
        prefix={<SearchOutlined />}
        value={searchValue}
        onChange={(e) => setSearchValue(e.target.value)}
        style={{ marginBottom: 16 }}
      />

      {/* 全选 */}
      <div style={{ marginBottom: 16 }}>
        <Checkbox
          checked={allChecked}
          indeterminate={someChecked}
          onChange={(e) => handleToggleAll(e.target.checked)}
        >
          全选
        </Checkbox>
        <span className="ml-4 text-gray-400">
          共 {localConfig.length} 列，显示 {localConfig.filter(c => c.visible).length} 列
        </span>
      </div>

      {/* 列配置列表 */}
      <div style={{ maxHeight: 400, overflowY: "auto" }}>
        <DndContext
          collisionDetection={closestCenter}
          onDragEnd={handleDragEnd}
        >
          <SortableContext
            items={filteredConfig.map(c => c.key)}
            strategy={verticalListSortingStrategy}
          >
            {filteredConfig.map((col) => (
              <SortableColumnItem
                key={col.key}
                config={col}
                onToggleVisible={(visible) =>
                  setLocalConfig(prev =>
                    prev.map(c => (c.key === col.key ? { ...c, visible } : c))
                  )
                }
                onWidthChange={(width) =>
                  setLocalConfig(prev =>
                    prev.map(c => (c.key === col.key ? { ...c, width: width || 0 } : c))
                  )
                }
              />
            ))}
          </SortableContext>
        </DndContext>
      </div>

      {/* 搜索无结果提示 */}
      {filteredConfig.length === 0 && (
        <div className="text-center py-8 text-gray-400">
          未找到匹配的列
        </div>
      )}
    </Modal>
  );
};
