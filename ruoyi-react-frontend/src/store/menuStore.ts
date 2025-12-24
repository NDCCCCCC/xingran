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
