/**
 * 布局提供者组件
 * 在应用根部包裹，提供布局功能
 */

import { useEffect, createContext, useContext, useMemo } from "react";
import type { FC, ReactNode } from "react";
import { useLayoutStore } from "@/store/layoutStore";

interface LayoutContextValue {
  layout: string;
  sidebarCollapsed: boolean;
}

const LayoutContext = createContext<LayoutContextValue>({
  layout: "classic",
  sidebarCollapsed: false,
});

export const useLayoutContext = () => useContext(LayoutContext);

const LayoutProvider: FC<{ children: ReactNode }> = ({ children }) => {
  const currentLayout = useLayoutStore((s) => s.currentLayout);
  const sidebarCollapsed = useLayoutStore((s) => s.sidebarCollapsed);
  const density = useLayoutStore((s) => s.density);

  useEffect(() => {
    // 设置data-layout属性
    document.documentElement.setAttribute("data-layout", currentLayout);
    document.documentElement.setAttribute("data-density", density);
  }, [currentLayout, density]);

  useEffect(() => {
    // 设置侧边栏折叠状态
    if (sidebarCollapsed) {
      document.documentElement.setAttribute("data-sidebar-collapsed", "true");
    } else {
      document.documentElement.removeAttribute("data-sidebar-collapsed");
    }
  }, [sidebarCollapsed]);

  const contextValue = useMemo(
    () => ({ layout: currentLayout, sidebarCollapsed }),
    [currentLayout, sidebarCollapsed]
  );

  return <LayoutContext.Provider value={contextValue}>{children}</LayoutContext.Provider>;
};

export default LayoutProvider;
