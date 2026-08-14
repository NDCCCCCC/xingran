/**
 * 表格操作按钮组件
 * 当按钮数量 >= 3 个时，所有按钮都收纳到下拉菜单中
 * 否则直接显示所有按钮
 */

import { Space, Button, Dropdown, Popconfirm } from "antd";
import { DownOutlined, DeleteOutlined } from "@ant-design/icons";
import type { MenuProps } from "antd";
import type { FC } from "react";
import { DELETE_CONFIRM } from "@/constants/buttonStyles";
import "@/components/shared/ActionButtons.css";

export interface ActionButton {
  key: string;
  label: string;
  icon?: React.ReactNode;
  onClick?: () => void;
  danger?: boolean;
  type?: "link" | "text" | "primary" | "default" | "dashed";
  disabled?: boolean;
  render?: () => React.ReactNode;
}

interface ActionButtonsProps {
  actions: ActionButton[];
  threshold?: number;
  size?: "small" | "middle" | "large";
}

const ActionButtons: FC<ActionButtonsProps> = ({ actions, threshold = 3, size = "small" }) => {
  if (actions.length === 0) {
    return null;
  }

  // 按钮数量 < 阈值，直接显示所有按钮
  if (actions.length < threshold) {
    return (
      <Space size="small">
        {actions.map((action) => (
          <div
            key={action.key}
            // 阻止点击冒泡到外层 <Table onRow.onClick>，避免触发行级副作用
            // （如工位页的"行点击切换展开子表"）
            onClick={(e) => e.stopPropagation()}
          >
            {action.render ? (
              action.render()
            ) : (
              <Button
                type={action.type ?? "link"}
                size={size}
                icon={action.icon}
                onClick={action.onClick}
                disabled={action.disabled}
                className={action.danger ? "action-btn-link-danger" : "action-btn-link"}
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
  const menuItems: MenuProps["items"] = actions.map((action) => {
    if (action.render) {
      return {
        key: action.key,
        label: <div onClick={(e) => e.stopPropagation()}>{action.render()}</div>,
      };
    }

    return {
      key: action.key,
      label: action.label,
      icon: action.icon,
      danger: action.danger,
      disabled: action.disabled,
      onClick: action.onClick,
      className: action.danger ? "action-dropdown-item-danger" : "action-dropdown-item",
    };
  });

  return (
    <Dropdown menu={{ items: menuItems }} trigger={["click"]}>
      <Button type="link" size={size} icon={<DownOutlined />} className="action-btn-link">
        操作
      </Button>
    </Dropdown>
  );
};

export default ActionButtons;

/**
 * 创建删除按钮 Action 配置
 * @param id 删除项的 ID
 * @param handleDelete 删除处理函数
 * @param options 配置选项
 * @returns 删除按钮 Action 配置
 */
export const createDeleteAction = <T extends string | number>(
  id: T,
  handleDelete: (id: T) => void | Promise<void>,
  options?: {
    /** 确认框标题 */
    title?: string;
    /** 按钮标签 */
    label?: string;
    /** 是否禁用 */
    disabled?: boolean;
    /** 是否使用 ghost 样式（红色边框+文字） */
    ghost?: boolean;
  }
): ActionButton => ({
  key: "delete",
  label: options?.label || "删除",
  icon: <DeleteOutlined />,
  danger: true,
  disabled: options?.disabled,
  render: () => (
    <Popconfirm
      title={options?.title || DELETE_CONFIRM.title}
      onConfirm={() => handleDelete(id)}
      okText={DELETE_CONFIRM.okText}
      cancelText={DELETE_CONFIRM.cancelText}
    >
      <Button
        type="link"
        danger
        ghost={options?.ghost ?? true}
        icon={<DeleteOutlined />}
        size="small"
        disabled={options?.disabled}
        className="action-btn-link-danger"
      >
        {options?.label || "删除"}
      </Button>
    </Popconfirm>
  ),
});

/**
 * 创建带自定义渲染的删除按钮 Action 配置
 * 用于需要完全自定义删除按钮渲染的场景
 * @param render 自定义渲染函数
 * @returns 删除按钮 Action 配置
 */
export const createCustomDeleteAction = (render: () => React.ReactNode): ActionButton => ({
  key: "delete",
  label: "删除",
  icon: <DeleteOutlined />,
  danger: true,
  render,
});
