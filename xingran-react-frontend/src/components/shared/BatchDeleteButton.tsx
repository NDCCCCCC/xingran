/**
 * 批量删除按钮组件
 * 统一的批量删除按钮，支持计数显示和二次确认
 */

import { Button, Popconfirm } from "antd";
import { DeleteOutlined } from "@ant-design/icons";
import type { FC } from "react";
import { BATCH_DELETE_CONFIRM } from "@/constants/buttonStyles";
import "./ActionButtons.css";

interface BatchDeleteButtonProps {
  /** 选中的数量 */
  selectedCount: number;
  /** 确认后的回调 */
  onConfirm: () => void | Promise<void>;
  /** 是否加载中 */
  loading?: boolean;
  /** 确认框标题 */
  confirmTitle?: string;
  /** 按钮禁用状态 */
  disabled?: boolean;
  /** 是否使用 ghost 样式（红色边框+文字） */
  ghost?: boolean;
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
}) => {
  if (selectedCount === 0) {
    return null;
  }

  return (
    <Popconfirm
      title={confirmTitle}
      onConfirm={onConfirm}
      okText={BATCH_DELETE_CONFIRM.okText}
      cancelText={BATCH_DELETE_CONFIRM.cancelText}
      disabled={disabled}
    >
      <Button
        danger
        ghost={ghost}
        icon={<DeleteOutlined />}
        loading={loading}
        disabled={disabled}
        className="action-btn-link-danger"
      >
        批量删除 ({selectedCount})
      </Button>
    </Popconfirm>
  );
};

export default BatchDeleteButton;
