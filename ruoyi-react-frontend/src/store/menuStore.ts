import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { getUserMenus, getAllUserMenus, getUserPermissions } from '@/lib/menuApi';
import type { Menu } from '@/types';

interface MenuState {
  menus: Menu[];
  allMenus: Menu[];
  permissions: string[];
  loading: boolean;
}

interface MenuActions {
  fetchMenus: () => Promise<void>;
  fetchPermissions: () => Promise<void>;
  setMenus: (menus: Menu[]) => void;
  clearMenus: () => void;
  setPermissions: (permissions: string[]) => void;
  refreshMenus: () => Promise<void>;
}

type MenuStore = MenuState & MenuActions;

const fetchMenuData = async () => {
  const [visibleMenus, allMenuList] = await Promise.all([
    getUserMenus(),
    getAllUserMenus(),
  ]);
  return { visibleMenus, allMenuList };
};

export const useMenuStore = create<MenuStore>()(
  persist(
    (set) => ({
      menus: [],
      allMenus: [],
      permissions: [],
      loading: false,

      fetchMenus: async () => {
        set({ loading: true });
        try {
          const { visibleMenus, allMenuList } = await fetchMenuData();
          set({ menus: visibleMenus, allMenus: allMenuList, loading: false });
        } catch (error) {
          set({ loading: false });
          throw error;
        }
      },

      fetchPermissions: async () => {
        try {
          const permissions = await getUserPermissions();
          set({ permissions });
        } catch (error) {
          set({ permissions: [] });
          throw error;
        }
      },

      setMenus: (menus) => {
        set({ menus });
      },

      clearMenus: () => {
        set({ menus: [], allMenus: [], permissions: [] });
      },

      setPermissions: (permissions) => {
        set({ permissions });
      },

      refreshMenus: async () => {
        set({ loading: true });
        try {
          const { visibleMenus, allMenuList } = await fetchMenuData();
          set({ menus: visibleMenus, allMenus: allMenuList, loading: false });
        } catch (error) {
          set({ loading: false });
          throw error;
        }
      },
    }),
    {
      name: 'menu-storage',
      version: 2,
      partialize: (state) => ({
        menus: state.menus,
        allMenus: state.allMenus,
      }),
    }
  )
);

export const refreshMenuCache = async () => {
  await useMenuStore.getState().refreshMenus();
};

export const clearMenuCache = () => {
  localStorage.removeItem('menu-storage');
  useMenuStore.getState().clearMenus();
};
