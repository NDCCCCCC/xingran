/**
 * 空数据态组件
 *
 * 接受 description / actionLabel / actionPath / icon props,
 * 内部用 AntD Empty 组件 + Button 包裹导航跳转。
 * 当 actionLabel 和 actionPath 同时存在时显示按钮,否则仅展示空状态描述。
 *
 * 使用场景:列表/查询结果为 0 时,提供"前往某页面"引导。
 */

import type { FC, ReactNode } from 'react';
import { Empty, Button, Space } from 'antd';
import { Link } from 'react-router-dom';

export interface EmptyStateWithActionProps {
  /** 空状态描述文案 */
  description: string;
  /** 按钮文案(可选) */
  actionLabel?: string;
  /** 按钮跳转路径(可选) */
  actionPath?: string;
  /** 自定义图标(可选,默认 AntD Empty 默认图标) */
  icon?: ReactNode;
}

const EmptyStateWithAction: FC<EmptyStateWithActionProps> = ({
  description,
  actionLabel,
  actionPath,
  icon,
}) => {
  const showAction =
    Boolean(actionLabel) &&
    typeof actionPath === 'string' &&
    actionPath.length > 0;

  return (
    <Empty
      image={icon ?? Empty.PRESENTED_IMAGE_SIMPLE}
      description={description}
      style={{ padding: '32px 0' }}
    >
      {showAction && (
        <Space>
          <Link to={actionPath}>
            <Button type="primary">{actionLabel}</Button>
          </Link>
        </Space>
      )}
    </Empty>
  );
};

export default EmptyStateWithAction;
