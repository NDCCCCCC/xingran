/**
 * 布局提供者组件
 * 在应用根部包裹，提供布局功能
 */

import React, { useEffect, createContext, useContext } from 'react';
import { useLayoutStore } from '@/store/layoutStore';

interface LayoutContextValue {
  layout: string;
  sidebarCollapsed: boolean;
}

const LayoutContext = createContext<LayoutContextValue>({
  layout: 'classic',
  sidebarCollapsed: false,
});

export const useLayoutContext = () => useContext(LayoutContext);

export const LayoutProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { currentLayout, sidebarCollapsed, density } = useLayoutStore();

  useEffect(() => {
    // 设置data-layout属性
    document.documentElement.setAttribute('data-layout', currentLayout);
    document.documentElement.setAttribute('data-density', density);
  }, [currentLayout, density]);

  useEffect(() => {
    // 设置侧边栏折叠状态
    if (sidebarCollapsed) {
      document.documentElement.setAttribute('data-sidebar-collapsed', 'true');
    } else {
      document.documentElement.removeAttribute('data-sidebar-collapsed');
    }
  }, [sidebarCollapsed]);

  return (
    <LayoutContext.Provider value={{ layout: currentLayout, sidebarCollapsed }}>
      {children}
    </LayoutContext.Provider>
  );
};
