/**
 * 布局状态管理
 * 使用Zustand进行布局切换和持久化
 */

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import React from 'react';
import type { LayoutType, LayoutConfig, DensityMode } from '@/types/layout';

interface LayoutState {
  currentLayout: LayoutType;
  sidebarCollapsed: boolean;
  density: DensityMode;
}

interface LayoutActions {
  setLayout: (layout: LayoutType) => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  toggleSidebar: () => void;
  setDensity: (density: DensityMode) => void;
  resetLayout: () => void;
}

type LayoutStore = LayoutState & LayoutActions;

// 布局配置
const layoutConfigs: Record<LayoutType, LayoutConfig> = {
  classic: {
    id: 'classic',
    name: '经典布局',
    description: '左侧导航+顶部栏+内容区',
    features: {
      sidebar: {
        collapsible: true,
        width: 256,
        collapsedWidth: 64,
        position: 'left',
        defaultCollapsed: false,
      },
      header: {
        fixed: true,
        height: 64,
        showBreadcrumb: true,
        showUserInfo: true,
      },
      tabs: {
        enabled: false,
        position: 'top',
        closable: false,
        draggable: false,
        persist: false,
      },
      content: {
        padding: '24px',
        centered: false,
        scrollable: true,
      },
    },
  },
  hybrid: {
    id: 'hybrid',
    name: '混合式布局',
    description: '可折叠侧边栏+多标签页系统',
    features: {
      sidebar: {
        collapsible: true,
        width: 240,
        collapsedWidth: 56,
        position: 'left',
        defaultCollapsed: false,
      },
      header: {
        fixed: true,
        height: 56,
        showBreadcrumb: false,
        showUserInfo: true,
      },
      tabs: {
        enabled: true,
        position: 'top',
        closable: true,
        draggable: true,
        persist: true,
      },
      content: {
        padding: '16px',
        centered: false,
        scrollable: true,
      },
    },
  },
  innovative: {
    id: 'innovative',
    name: '创新布局',
    description: '创新导航方式+模块化面板',
    features: {
      sidebar: {
        collapsible: false,
        width: 80,
        collapsedWidth: 80,
        position: 'left',
        defaultCollapsed: false,
      },
      header: {
        fixed: false,
        height: 0,
        showBreadcrumb: false,
        showUserInfo: false,
      },
      tabs: {
        enabled: true,
        position: 'bottom',
        closable: true,
        draggable: false,
        persist: true,
      },
      content: {
        padding: '0',
        centered: true,
        scrollable: true,
      },
    },
  },
};

export const useLayoutStore = create<LayoutStore>()(
  persist(
    (set, get) => ({
      // 初始状态 - 从localStorage同步读取，避免首次渲染闪烁
      currentLayout: (() => {
        try {
          const stored = localStorage.getItem('layout-storage');
          if (stored) {
            const parsed = JSON.parse(stored);
            return parsed.state?.currentLayout || 'classic';
          }
        } catch (e) {
          console.error('Failed to read layout from localStorage:', e);
        }
        return 'classic';
      })(),
      sidebarCollapsed: (() => {
        try {
          const stored = localStorage.getItem('layout-storage');
          if (stored) {
            const parsed = JSON.parse(stored);
            return parsed.state?.sidebarCollapsed ?? false;
          }
        } catch (e) {
          console.error('Failed to read sidebarCollapsed from localStorage:', e);
        }
        return false;
      })(),
      density: (() => {
        try {
          const stored = localStorage.getItem('layout-storage');
          if (stored) {
            const parsed = JSON.parse(stored);
            return parsed.state?.density || 'comfortable';
          }
        } catch (e) {
          console.error('Failed to read density from localStorage:', e);
        }
        return 'comfortable';
      })(),

      // 设置布局
      setLayout: (layout: LayoutType) => {
        set({
          currentLayout: layout,
        });

        // 设置data-layout属性
        document.documentElement.setAttribute('data-layout', layout);
      },

      // 设置侧边栏折叠状态
      setSidebarCollapsed: (collapsed: boolean) => {
        set({ sidebarCollapsed: collapsed });
      },

      // 切换侧边栏
      toggleSidebar: () => {
        set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed }));
      },

      // 设置密度模式
      setDensity: (density: DensityMode) => {
        set({ density: density });
        document.documentElement.setAttribute('data-density', density);
      },

      // 重置布局
      resetLayout: () => {
        get().setLayout('classic');
        get().setSidebarCollapsed(false);
        get().setDensity('comfortable');
      },
    }),
    {
      name: 'layout-storage',
      partialize: (state) => ({
        currentLayout: state.currentLayout,
        sidebarCollapsed: state.sidebarCollapsed,
        density: state.density,
        // 不持久化 _hasHydrated
      }),
      onRehydrateStorage: () => (state) => {
        if (state) {
          document.documentElement.setAttribute('data-layout', state.currentLayout);
          document.documentElement.setAttribute('data-density', state.density);
        }
      },
    }
  )
);

/**
 * 布局Hook
 * 提供更方便的布局访问和操作
 */
export function useLayout() {
  const { currentLayout, sidebarCollapsed, density, setLayout, toggleSidebar, setDensity } = useLayoutStore();

  // 同步data-layout和data-density属性
  React.useEffect(() => {
    document.documentElement.setAttribute('data-layout', currentLayout);
    document.documentElement.setAttribute('data-density', density);
  }, [currentLayout, density]);

  const layoutConfig = layoutConfigs[currentLayout];

  return {
    layout: currentLayout,
    layoutConfig,
    sidebarCollapsed,
    density,
    setLayout,
    toggleSidebar,
    setDensity,
    isClassic: currentLayout === 'classic',
    isHybrid: currentLayout === 'hybrid',
    isInnovative: currentLayout === 'innovative',
    isCompact: density === 'compact',
    isComfortable: density === 'comfortable',
    isSpacious: density === 'spacious',
  };
}

/**
 * 导出布局配置
 */
export { layoutConfigs };
