import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { getUserMenus } from '@/lib/menuApi';
import type { Menu } from '@/types';

interface MenuState {
  menus: Menu[];
  permissions: string[];
  loading: boolean;
}

interface MenuActions {
  fetchMenus: () => Promise<void>;
  setMenus: (menus: Menu[]) => void;
  clearMenus: () => void;
  setPermissions: (permissions: string[]) => void;
  refreshMenus: () => Promise<void>; // 新增：刷新菜单（清除缓存并重新获取）
}

export const useMenuStore = create<MenuState & MenuActions>()(
  persist(
    (set) => ({
      // 初始状态
      menus: [],
      permissions: [],
      loading: false,

      // 获取用户菜单
      fetchMenus: async () => {
        set({ loading: true });
        try {
          const menus = await getUserMenus();
          set({ menus, loading: false });
        } catch (error) {
          set({ loading: false });
          throw error;
        }
      },

      // 设置菜单
      setMenus: (menus: Menu[]) => {
        set({ menus });
      },

      // 清除菜单
      clearMenus: () => {
        set({ menus: [], permissions: [] });
      },

      // 设置权限
      setPermissions: (permissions: string[]) => {
        set({ permissions });
      },

      // 刷新菜单：清除缓存并重新获取
      refreshMenus: async () => {
        // 清空当前菜单数据
        set({ menus: [], permissions: [] });
        // 重新从服务器获取
        try {
          const menus = await getUserMenus();
          set({ menus, loading: false });
        } catch (error) {
          set({ loading: false });
          throw error;
        }
      },
    }),
    {
      name: 'menu-storage',
      partialize: (state) => ({
        menus: state.menus,
        permissions: state.permissions,
      }),
    }
  )
);

// 刷新菜单缓存的辅助函数
// 注意：只清除菜单缓存并重新获取，不影响用户认证信息
export const refreshMenuCache = async () => {
  const menuStore = useMenuStore.getState();
  await menuStore.refreshMenus();
};

// 清除菜单缓存的辅助函数（兼容旧代码）
// 注意：只清除菜单缓存，不影响用户认证信息
export const clearMenuCache = () => {
  // 清除 localStorage 中的菜单缓存
  localStorage.removeItem('menu-storage');
  // 同时清除 Zustand 内部存储的菜单数据
  const menuStore = useMenuStore.getState();
  menuStore.clearMenus();
};
