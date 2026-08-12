/**
 * BaseEditModal - 通用编辑模态框
 *
 * 基于 AntD Modal 的薄包装，用 React.memo 包装以避免父组件重渲染时无谓地
 * 重建 Modal 子树。适用于资产、VDI、用户等列表页面的编辑/新建弹窗。
 *
 * 父组件应当：
 * - 通过 useCallback 稳定 onOk / onCancel 引用
 * - 通过 useMemo 稳定 props 对象（可选，但推荐）
 */

import { memo, type ReactNode } from "react";
import { Modal } from "antd";

export interface BaseEditModalProps {
  open: boolean;
  title: string;
  onOk: () => void;
  onCancel: () => void;
  /** 确认按钮 loading 状态 */
  confirmLoading?: boolean;
  /** Modal 宽度（默认 520） */
  width?: number;
  /** 底部按钮文案 */
  okText?: string;
  cancelText?: string;
  /** 是否允许点击蒙层关闭（默认 true） */
  maskClosable?: boolean;
  children: ReactNode;
}

function BaseEditModalImpl({
  open,
  title,
  onOk,
  onCancel,
  confirmLoading = false,
  width = 520,
  okText = "确定",
  cancelText = "取消",
  maskClosable = true,
  children,
}: BaseEditModalProps) {
  return (
    <Modal
      open={open}
      title={title}
      onOk={onOk}
      onCancel={onCancel}
      confirmLoading={confirmLoading}
      width={width}
      okText={okText}
      cancelText={cancelText}
      maskClosable={maskClosable}
      destroyOnHidden
    >
      {children}
    </Modal>
  );
}

export const BaseEditModal = memo(BaseEditModalImpl);
BaseEditModal.displayName = "BaseEditModal";

export default BaseEditModal;
