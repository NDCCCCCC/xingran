/**
 * Phase 65 契约 — 危险删除唯一声明式入口（UI-SPEC §4）
 *
 * 内部直接复用命令式 helper confirmDelete()（shared/confirmDelete.ts），
 * 不自建第二套 Modal 逻辑。默认渲染 danger 删除按钮（DeleteOutlined +
 * action-btn-link-danger 类，ActionButtons 惯例）；children 替换按钮内容；
 * disabled=true 或 onDelete 缺失时按钮禁用。
 * 行为契约（E1/E6 探针）：确认中 loading 防重（onOk Promise 语义）、
 * 失败 Modal 保持打开可重试（helper 内 rethrow）。
 */
import { Button } from "antd";
import { DeleteOutlined } from "@ant-design/icons";
import type { ConfirmDeleteOptions } from "./confirmDelete";
import { confirmDelete } from "./confirmDelete";

export interface ConfirmDeleteProps extends Omit<ConfirmDeleteOptions, "onDelete"> {
  /** 确认后执行的删除动作；未传时按钮禁用 */
  onDelete?: () => Promise<void>;
  /** 自定义按钮内容，默认「删除」文字 + DeleteOutlined */
  children?: React.ReactNode;
  /** 是否禁用按钮（默认 false） */
  disabled?: boolean;
}

/** 危险删除按钮（声明式形态） */
export function ConfirmDelete({
  target,
  onDelete,
  children,
  title,
  extra,
  disabled,
  actionLabel,
  actionContext,
  contentText,
  successText,
}: ConfirmDeleteProps) {
  const isDisabled = disabled || typeof onDelete !== "function";

  const handleClick = () => {
    if (isDisabled || !onDelete) return;
    confirmDelete({
      target,
      onDelete,
      title,
      extra,
      actionLabel,
      actionContext,
      contentText,
      successText,
    });
  };

  return (
    <Button
      type="link"
      danger
      icon={<DeleteOutlined />}
      className="action-btn-link-danger"
      // Q1：可访问名与弹窗标题解耦——title 只作确认弹窗标题，不流入 aria-label
      aria-label={`删除${target}`}
      disabled={isDisabled}
      onClick={handleClick}
    >
      {children ?? "删除"}
    </Button>
  );
}
