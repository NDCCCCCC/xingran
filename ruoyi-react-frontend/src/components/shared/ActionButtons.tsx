/**
 * 表格操作按钮组件
 * 当按钮数量 >= 3 个时，所有按钮都收纳到下拉菜单中
 * 否则直接显示所有按钮
 */

import { Space, Button, Dropdown } from 'antd';
import { DownOutlined } from '@ant-design/icons';
import type { MenuProps } from 'antd';

export interface ActionButton {
  key: string;
  label: string;
  icon?: React.ReactNode;
  onClick?: () => void;
  danger?: boolean;
  type?: 'link' | 'text' | 'primary' | 'default' | 'dashed';
  disabled?: boolean;
  render?: () => React.ReactNode; // 自定义渲染（如 Popconfirm）
}

interface ActionButtonsProps {
  actions: ActionButton[];
  threshold?: number; // 触发下拉菜单的按钮数量阈值，默认为3
  size?: 'small' | 'middle' | 'large';
}

const ActionButtons: React.FC<ActionButtonsProps> = ({
  actions,
  threshold = 3,
  size = 'small',
}) => {
  if (actions.length === 0) {
    return null;
  }

  // 如果按钮数量 < 阈值，直接显示所有按钮
  if (actions.length < threshold) {
    return (
      <Space size="small">
        {actions.map((action) => (
          <div key={action.key}>
            {action.render ? (
              action.render()
            ) : (
              <Button
                type={action.type || 'link'}
                size={size}
                icon={action.icon}
                onClick={action.onClick}
                danger={action.danger}
                disabled={action.disabled}
              >
                {action.label}
              </Button>
            )}
          </div>
        ))}
      </Space>
    );
  }

  // 按钮数量 >= 阈值，所有按钮都放到下拉菜单中
  const menuItems: MenuProps['items'] = actions.map((action) => {
    if (action.render) {
      // 对于自定义渲染的按钮（如 Popconfirm），需要特殊处理
      return {
        key: action.key,
        label: (
          <div onClick={(e) => e.stopPropagation()}>
            {action.render()}
          </div>
        ),
      };
    }

    return {
      key: action.key,
      label: action.label,
      icon: action.icon,
      danger: action.danger,
      disabled: action.disabled,
      onClick: action.onClick,
    };
  });

  return (
    <Dropdown menu={{ items: menuItems }} trigger={['click']}>
      <Button type="link" size={size} icon={<DownOutlined />}>
        操作
      </Button>
    </Dropdown>
  );
};

export default ActionButtons;
