/**
 * Widget - 仪表盘 Widget 顶层容器
 *
 * 包裹现有的 `WidgetRenderer`，用 React.memo 包装以避免父组件（DashboardGrid）
 * 重渲染时无谓地重建所有 widget 子树。仅当 widget 配置对象变化时才重新渲染。
 *
 * 父组件应当：
 * - 通过 useMemo 稳定 widget props 对象（强烈推荐，否则 memo 不生效）
 * - 通过 useCallback 稳定 onEdit / onDelete 引用
 */

import { memo } from "react";
import type { WidgetConfig } from "@/types/dashboard";
import { WidgetRenderer } from "./widgets/WidgetRenderer";

export interface WidgetProps {
  widget: WidgetConfig;
  onEdit?: () => void;
  onDelete?: () => void;
}

function WidgetImpl({ widget, onEdit, onDelete }: WidgetProps) {
  return <WidgetRenderer widget={widget} onEdit={onEdit} onDelete={onDelete} />;
}

export const Widget = memo(WidgetImpl);
Widget.displayName = "Widget";

export default Widget;
