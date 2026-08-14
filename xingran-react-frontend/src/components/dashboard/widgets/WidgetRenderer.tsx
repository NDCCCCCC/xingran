/**
 * WidgetRenderer - Widget 渲染器
 *
 * 根据Widget类型动态渲染对应的组件
 */

import { Suspense } from "react";
import { Spin } from "antd";
import type { WidgetConfig } from "@/types/dashboard";
import { widgetRegistry } from "./configs/widgetRegistry";
import { BaseWidget } from "./base/BaseWidget";

interface WidgetRendererProps {
  widget: WidgetConfig;
  onEdit?: () => void;
  onDelete?: () => void;
}

/**
 * Widget 渲染器组件
 */
export const WidgetRenderer: React.FC<WidgetRendererProps> = ({ widget, onEdit, onDelete }) => {
  const WidgetComponent = widgetRegistry[widget.type]?.component;

  if (!WidgetComponent) {
    return <div>未知的Widget类型: {widget.type}</div>;
  }

  // 准备传递给Widget组件的props
  // 注意：Widget组件（如ProgressWidget）已经包装了BaseWidget
  // 我们只需要传递操作回调，其他由Widget组件自行处理
  const componentProps = {
    widget,
    display: widget.display,
    onEdit,
    onDelete,
  };

  // 直接渲染Widget组件
  return (
    <Suspense fallback={<Spin />}>
      <WidgetComponent {...(componentProps as any)} />
    </Suspense>
  );
};

export default WidgetRenderer;
