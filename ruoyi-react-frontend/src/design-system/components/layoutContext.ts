/**
 * 布局 Context（v1.24 P82 自 LayoutProvider.tsx 拆出：
 * react-refresh 要求组件文件只导出组件，hook 与组件同文件会破坏 Fast Refresh 边界）
 */
import { createContext, useContext } from "react";

export interface LayoutContextValue {
  layout: string;
  sidebarCollapsed: boolean;
}

export const LayoutContext = createContext<LayoutContextValue>({
  layout: "classic",
  sidebarCollapsed: false,
});

export const useLayoutContext = () => useContext(LayoutContext);
