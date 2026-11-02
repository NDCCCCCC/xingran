/**
 * Widget 类型助手函数
 * 用于处理 Widget 组件的类型转换
 */

import React from "react";
import type { WidgetConfig } from "../dashboard";

/**
 * Widget 基础 Props 接口
 * 所有 Widget 组件应实现此接口
 */
export interface WidgetBaseProps {
  /** Widget 配置 */
  widget: WidgetConfig;
  /** 显示配置 */
  display: Record<string, unknown>;
  /** 编辑回调 */
  onEdit?: () => void;
  /** 删除回调 */
  onDelete?: () => void;
}

/**
 * Widget 组件类型（统一签名）
 * 使用更宽松的类型以兼容各种 Widget 实现
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type WidgetComponentType = React.ComponentType<any>;

/**
 * 安全地将 Widget 组件转换为通用类型
 * 解决 React.lazy 加载组件的类型兼容问题
 *
 * @example
 * ```typescript
 * component: lazy(() => import('./StatCardWidget').then(m => ({
 *   default: asWidgetComponent(m.StatCardWidget)
 * }))),
 * ```
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function asWidgetComponent<T extends React.ComponentType<any>>(
  component: T
): WidgetComponentType {
  return component as WidgetComponentType;
}

/**
 * 创建类型安全的懒加载 Widget
 * 封装 React.lazy + 类型转换的常见模式
 *
 * @example
 * ```typescript
 * component: createLazyWidget(() => import('./StatCardWidget'), 'StatCardWidget'),
 * ```
 */
export function createLazyWidget<T extends React.ComponentType<unknown>>(
  importFn: () => Promise<{ [key: string]: T }>,
  exportName: string
): React.LazyExoticComponent<WidgetComponentType> {
  return React.lazy(async () => {
    const module = await importFn();
    return { default: module[exportName] as WidgetComponentType };
  });
}
