/**
 * 按钮样式常量配置
 * 统一项目中各类按钮的样式规范
 */

import React from "react";
import { DeleteOutlined, StopOutlined, CheckCircleOutlined } from "@ant-design/icons";

/**
 * 按钮类型枚举
 */
export enum ButtonType {
  /** 默认按钮 */
  Default = "default",
  /** 主要按钮 */
  Primary = "primary",
  /** 链接按钮 */
  Link = "link",
  /** 文本按钮 */
  Text = "text",
  /** 虚线按钮 */
  Dashed = "dashed",
}

/**
 * 按钮尺寸枚举
 */
export enum ButtonSize {
  /** 小尺寸 */
  Small = "small",
  /** 中等尺寸 */
  Middle = "middle",
  /** 大尺寸 */
  Large = "large",
}

/**
 * 统一按钮样式配置
 */
export const BUTTON_STYLES = {
  /** 删除按钮样式 - 使用 ghost 模式显示红色边框和文字 */
  DELETE: {
    type: ButtonType.Link,
    size: ButtonSize.Small,
    ghost: true,
    danger: true,
  },
  /** 批量删除按钮样式 */
  BATCH_DELETE: {
    ghost: true,
    danger: true,
  },
  /** 状态切换按钮样式 */
  STATUS_TOGGLE: {
    type: ButtonType.Link,
    size: ButtonSize.Small,
  },
  /** 编辑按钮样式 */
  EDIT: {
    type: ButtonType.Link,
    size: ButtonSize.Small,
  },
} as const;

/**
 * 删除确认框配置
 */
export const DELETE_CONFIRM = {
  title: "确定要删除吗？",
  okText: "确定",
  cancelText: "取消",
};

/**
 * 批量删除确认框配置
 */
export const BATCH_DELETE_CONFIRM = {
  title: "确定要批量删除选中项吗？",
  okText: "确定",
  cancelText: "取消",
};

/**
 * 状态切换图标配置
 */
export const STATUS_ICONS = {
  /** 停用图标 */
  STOP: <StopOutlined />,
  /** 启用图标 */
  CHECK: <CheckCircleOutlined />,
} as const;
