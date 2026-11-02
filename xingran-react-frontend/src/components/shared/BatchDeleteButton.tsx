/**
 * 批量删除按钮组件
 * 统一的批量删除按钮，支持计数显示和二次确认
 *
 * Phase 65：确认路径内部切换到唯一入口 confirmDelete()（UI-SPEC §4 行为契约 1），
 * 移除裸二次确认；props 对外零变化。
 */

import { Button } from "antd";
import { DeleteOutlined } from "@ant-design/icons";
import type { FC } from "react";
import { BATCH_DELETE_CONFIRM } from "@/constants/buttonStyles";
import { confirmDelete } from "./confirmDelete";
import "./ActionButtons.css";

interface BatchDeleteButtonProps {
  /** 选中的数量 */
  selectedCount: number;
  /** 确认后的回调 */
  onConfirm: () => void | Promise<void>;
  /** 是否加载中 */
  loading?: boolean;
  /** 确认框标题（默认值随常量升级为「确认批量删除」，与 helper 批量语义一致） */
  confirmTitle?: string;
  /** 按钮禁用状态 */
  disabled?: boolean;
  /** 是否使用 ghost 样式（红色边框+文字） */
  ghost?: boolean;
  /** 对象类别（如 "资产"）；确认文案派生为 `选中的 {N} 项{target}`；缺省行为逐字不变（68.1 minor #2） */
  target?: string;
}

/**
 * 批量删除按钮组件
 * 当 selectedCount 为 0 时不显示按钮
 */
export const BatchDeleteButton: FC<BatchDeleteButtonProps> = ({
  selectedCount,
  onConfirm,
  loading = false,
  confirmTitle = BATCH_DELETE_CONFIRM.title,
  disabled = false,
  ghost = true,
  target,
}) => {
  if (selectedCount === 0) {
    return null;
  }

  const handleClick = () => {
    if (disabled) return;
    confirmDelete({
      target: target ? `选中的 ${selectedCount} 项${target}` : `选中的 ${selectedCount} 项`,
      onDelete: async () => {
        await onConfirm();
      },
      title: confirmTitle,
      actionContext: "批量删除",
    });
  };

  return (
    <Button
      danger
      ghost={ghost}
      icon={<DeleteOutlined />}
      loading={loading}
      disabled={disabled}
      className="action-btn-link-danger"
      onClick={handleClick}
    >
      批量删除 ({selectedCount})
    </Button>
  );
};

export default BatchDeleteButton;
