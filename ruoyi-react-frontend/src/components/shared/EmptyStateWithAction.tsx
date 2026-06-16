/**
 * 空数据态组件
 *
 * 接受 description / actionLabel / actionPath / icon props,
 * 内部用 AntD Empty 组件 + Button 包裹导航跳转。
 * 当 actionLabel 和 actionPath 同时存在时显示按钮,否则仅展示空状态描述。
 *
 * 使用场景:列表/查询结果为 0 时,提供"前往某页面"引导。
 */

import type { FC, ReactNode } from "react";
import { Empty, Button, Space } from "antd";
import { Link } from "react-router-dom";

export interface EmptyStateWithActionProps {
  /** 空状态标题(可选) */
  title?: string;
  /** 空状态描述文案 */
  description: string;
  /** 按钮文案(可选) */
  actionLabel?: string;
  /** 按钮跳转路径(可选) */
  actionPath?: string;
  /** 按钮点击回调(可选,二选一:actionPath 用于路由跳转,onAction 用于命令式动作) */
  onAction?: () => void;
  /** 自定义图标(可选,默认 AntD Empty 默认图标) */
  icon?: ReactNode;
}

const EmptyStateWithAction: FC<EmptyStateWithActionProps> = ({
  title,
  description,
  actionLabel,
  actionPath,
  onAction,
  icon,
}) => {
  const showAction =
    Boolean(actionLabel) &&
    (typeof actionPath === "string" && actionPath.length > 0 || typeof onAction === "function");

  return (
    <Empty
      image={icon ?? Empty.PRESENTED_IMAGE_SIMPLE}
      description={
        title ? (
          <div>
            <div style={{ fontSize: 16, fontWeight: 500, marginBottom: 4 }}>{title}</div>
            <div>{description}</div>
          </div>
        ) : (
          description
        )
      }
      style={{ padding: "32px 0" }}
    >
      {showAction && actionLabel && (
        <Space>
          {typeof actionPath === "string" && actionPath.length > 0 ? (
            <Link to={actionPath}>
              <Button type="primary">{actionLabel}</Button>
            </Link>
          ) : (
            <Button type="primary" onClick={onAction}>{actionLabel}</Button>
          )}
        </Space>
      )}
    </Empty>
  );
};

export default EmptyStateWithAction;
