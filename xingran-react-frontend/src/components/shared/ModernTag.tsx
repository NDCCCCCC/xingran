/**
 * 现代化标签组件
 * 提供更美观、现代的标签样式
 */

import { Tag } from "antd";
import type { TagProps } from "antd";
import { CheckCircleOutlined, StopOutlined, ClockCircleOutlined } from "@ant-design/icons";
import type { FC } from "react";

/** 状态类型 */
export type StatusType = "success" | "error" | "warning" | "default" | "processing";

/** 状态标签配置 */
const STATUS_CONFIG: Record<
  StatusType,
  { className: string; icon?: FC<{ style?: React.CSSProperties }>; label: string }
> = {
  success: { className: "modern-tag-success", icon: CheckCircleOutlined, label: "正常" },
  error: { className: "modern-tag-error", icon: StopOutlined, label: "停用" },
  warning: { className: "modern-tag-warning", icon: ClockCircleOutlined, label: "警告" },
  default: { className: "modern-tag-default", label: "默认" },
  processing: { className: "modern-tag-processing", label: "进行中" },
};

/** ModernTagProps 接口 */
interface ModernTagProps extends Omit<TagProps, "color"> {
  /** 状态类型 */
  status?: StatusType;
  /** 自定义文本 */
  children?: string;
  /** 是否显示图标 */
  showIcon?: boolean;
  /** 是否使用现代化样式 */
  modern?: boolean;
}

/**
 * 现代化状态标签组件
 * 默认使用现代化样式，提供更柔和的视觉效果
 */
export const ModernTag: FC<ModernTagProps> = ({
  status = "default",
  children,
  showIcon = false,
  modern = true,
  ...props
}) => {
  const config = STATUS_CONFIG[status];
  const Icon = config.icon;
  const text = children || config.label;

  if (!modern) {
    return (
      <Tag color={status} {...props}>
        {text}
      </Tag>
    );
  }

  return (
    <Tag className={config.className} {...props}>
      {showIcon && Icon && <Icon style={{ marginRight: "4px", fontSize: "12px" }} />}
      {text}
    </Tag>
  );
};

/**
 * 渲染状态标签的便捷函数
 * @param status 状态值 (0=正常/启用, 1=停用/禁用)
 * @param config 可选配置
 */
export function renderStatusTag(
  status: number,
  config?: {
    /** 正常状态的文本 */
    normalText?: string;
    /** 停用状态的文本 */
    stopText?: string;
    /** 是否显示图标 */
    showIcon?: boolean;
    /** 是否使用现代化样式 */
    modern?: boolean;
  }
) {
  const { normalText = "正常", stopText = "停用", showIcon = true, modern = true } = config || {};

  return (
    <ModernTag status={status === 0 ? "success" : "error"} showIcon={showIcon} modern={modern}>
      {status === 0 ? normalText : stopText}
    </ModernTag>
  );
}

export default ModernTag;
