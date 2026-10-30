/**
 * 表格操作按钮组件
 * 当按钮数量 >= 3 个时，所有按钮都收纳到下拉菜单中
 * 否则直接显示所有按钮
 */

import { Space, Button, Dropdown } from "antd";
import { DownOutlined, DeleteOutlined } from "@ant-design/icons";
import type { MenuProps } from "antd";
import { useMemo, type FC } from "react";
import { DELETE_CONFIRM } from "@/constants/buttonStyles";
import { ConfirmDelete } from "./ConfirmDelete.tsx"; // 显式扩展名：Windows 大小写不敏感解析会命中 confirmDelete.ts
import { useActionPermission } from "./useActionPermission";
import "@/components/shared/ActionButtons.css";

export interface ActionButton {
  key: string;
  label: string;
  icon?: React.ReactNode;
  onClick?: () => void;
  danger?: boolean;
  type?: "link" | "text" | "primary" | "default" | "dashed";
  disabled?: boolean;
  /**
   * v1.24 PERM-01（可选字段）：权限码。
   * 不传 = 不做权限判定（保持既有调用方行为，v1.19 冻结交互面不变）。
   * 传了则由组件内的三态门控（useActionPermission）处理：
   * 权限就绪且不含该码 → 不渲染；权限未就绪 → 渲染但 disabled + 提示。
   */
  permission?: string;
  render?: () => React.ReactNode;
}

interface ActionButtonsProps {
  actions: ActionButton[];
  threshold?: number;
  size?: "small" | "middle" | "large";
}

const ActionButtons: FC<ActionButtonsProps> = ({ actions, threshold = 3, size = "small" }) => {
  // v1.24 PERM-01/02：逐按钮的门控决策。
  // 未传 permission 的按钮（既有 45 个调用点）决策恒为「可见 + 不禁用」，
  // 对外形状与既有行为完全不变。
  const gate = useActionPermission();
  const gated = useMemo(
    () => actions.map((action) => ({ action, decision: gate.decide(action.permission) })),
    [actions, gate]
  );
  const visible = gated.filter(({ decision }) => decision.visible);

  if (visible.length === 0) {
    return null;
  }

  // 按钮数量 < 阈值，直接显示所有按钮
  if (visible.length < threshold) {
    return (
      <Space size="small">
        {visible.map(({ action, decision }) => (
          <div
            key={action.key}
            // 阻止点击冒泡到外层 <Table onRow.onClick>，避免触发行级副作用
            // （如工位页的"行点击切换展开子表"）
            onClick={(e) => e.stopPropagation()}
          >
            {action.render ? (
              decision.disabled ? (
                // 未就绪：渲染但不可交互 + 提示（与普通按钮的 disabled 语义一致）
                <span
                  title={decision.hint}
                  aria-disabled="true"
                  style={{ pointerEvents: "none", opacity: 0.6 }}
                >
                  {action.render()}
                </span>
              ) : (
                action.render()
              )
            ) : (
              <Button
                type={action.type ?? "link"}
                size={size}
                icon={action.icon}
                onClick={action.onClick}
                disabled={action.disabled || decision.disabled}
                className={action.danger ? "action-btn-link-danger" : "action-btn-link"}
                aria-label={decision.hint ?? action.label}
                title={decision.hint ?? action.label}
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
  const menuItems: MenuProps["items"] = visible.map(({ action, decision }) => {
    if (action.render) {
      const rendered = decision.disabled ? (
        // 未就绪：渲染但不可交互 + 提示
        <span
          title={decision.hint}
          aria-disabled="true"
          style={{ pointerEvents: "none", opacity: 0.6 }}
        >
          {action.render()}
        </span>
      ) : (
        action.render()
      );
      return {
        key: action.key,
        label: <div onClick={(e) => e.stopPropagation()}>{rendered}</div>,
      };
    }

    return {
      key: action.key,
      // 未就绪时把提示挂到菜单项上（title/aria）；已就绪时保持既有纯文本形态
      label: decision.hint ? (
        <span title={decision.hint} aria-label={decision.hint}>
          {action.label}
        </span>
      ) : (
        action.label
      ),
      icon: action.icon,
      danger: action.danger,
      disabled: action.disabled || decision.disabled,
      onClick: action.onClick,
      className: action.danger ? "action-dropdown-item-danger" : "action-dropdown-item",
    };
  });

  return (
    <Dropdown menu={{ items: menuItems }} trigger={["click"]}>
      <Button
        type="link"
        size={size}
        icon={<DownOutlined />}
        className="action-btn-link"
        aria-label="操作"
        title="操作"
      >
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
// eslint-disable-next-line react-refresh/only-export-components -- ActionButtons 对外形状 v1.19 冻结（T1 契约，禁改 import 路径/导出形状），只登记不拆
export const createDeleteAction = <T extends string | number>(
  id: T,
  handleDelete: (id: T) => void | Promise<void>,
  options?: {
    /** 确认框标题 */
    title?: string;
    /**
     * 删除对象名（供确认文案「将删除{target}…」与可访问名「删除{target}」）
     * Phase 68.1：与按钮文案解耦——传 target 不会改变按钮上的「删除」文字。
     */
    target?: string;
    /**
     * @deprecated Phase 68.1：请不要再用 label 传对象名。
     * label 语义是「按钮标签」，传对象名会让行内按钮文字变成记录名（视觉回归）。
     * 传对象名请用 `target`；label 仅用于确实需要自定义按钮文字的极少数场景。
     */
    label?: string;
    /** 是否禁用 */
    disabled?: boolean;
    /** 是否使用 ghost 样式（红色边框+文字） */
    ghost?: boolean;
    /**
     * v1.24 PERM-01（可选）：权限码。
     * 不传 = 不做权限判定（保持既有调用方行为）；传了则由 ActionButtons 的
     * 三态门控处理（就绪且无权限 → 不渲染；未就绪 → 渲染但 disabled + 提示）。
     */
    permission?: string;
  }
): ActionButton => ({
  key: "delete",
  label: options?.label || "删除",
  icon: <DeleteOutlined />,
  danger: true,
  disabled: options?.disabled,
  permission: options?.permission,
  // Phase 65：render 内部切换到唯一声明式入口 ConfirmDelete（不再使用裸二次确认组件）。
  // 降级注记：原 ghost 视觉（红边框）不再支持——ConfirmDelete 呈现 type=link danger；
  // title 语义保留（覆盖确认标题，默认值消费升级后的 DELETE_CONFIRM.title="确认删除"）。
  // Phase 68.1（Q1/R1）：target 承载对象名，按钮文字固定「删除」；aria-label 由
  // ConfirmDelete 内建为「删除{target}」，与确认弹窗标题解耦。
  render: () => (
    <ConfirmDelete
      target={options?.target || "选中项"}
      onDelete={async () => {
        await handleDelete(id);
      }}
      title={options?.title || DELETE_CONFIRM.title}
      disabled={options?.disabled}
    >
      {options?.label || "删除"}
    </ConfirmDelete>
  ),
});

/**
 * 创建带自定义渲染的删除按钮 Action 配置
 * 用于需要完全自定义删除按钮渲染的场景
 * @param render 自定义渲染函数
 * @returns 删除按钮 Action 配置
 */
// eslint-disable-next-line react-refresh/only-export-components -- ActionButtons 对外形状 v1.19 冻结（T1 契约，禁改 import 路径/导出形状），只登记不拆
export const createCustomDeleteAction = (render: () => React.ReactNode): ActionButton => ({
  key: "delete",
  label: "删除",
  icon: <DeleteOutlined />,
  danger: true,
  render,
});
