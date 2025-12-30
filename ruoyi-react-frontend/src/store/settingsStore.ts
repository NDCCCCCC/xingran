import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { UserPreferences } from '@/types';
import { getUserPreferences, updateUserPreferences } from '@/lib/profileApi';

interface SettingsState {
  preferences: UserPreferences;
  loading: boolean;
}

interface SettingsActions {
  loadPreferences: () => Promise<void>;
  updatePreferences: (prefs: Partial<UserPreferences>) => Promise<void>;
  setTheme: (theme: 'light' | 'dark') => void;
  setPageSize: (size: number) => void;
}

const defaultPreferences: UserPreferences = {
  theme: 'light',
  language: 'zh-CN',
  pageSize: 10,
  sidebarCollapsed: false,
};

export const useSettingsStore = create<SettingsState & SettingsActions>()(
  persist(
    (set, get) => ({
      // 初始状态
      preferences: defaultPreferences,
      loading: false,

      // 加载用户设置
      loadPreferences: async () => {
        set({ loading: true });
        try {
          const prefs = await getUserPreferences();
          set({ preferences: prefs, loading: false });
        } catch (error) {
          console.error('加载用户设置失败:', error);
          set({ loading: false });
        }
      },

      // 更新用户设置
      updatePreferences: async (newPrefs) => {
        const { preferences } = get();
        const updatedPrefs = { ...preferences, ...newPrefs };

        try {
          await updateUserPreferences(updatedPrefs);
          set({ preferences: updatedPrefs });
        } catch (error) {
          console.error('更新用户设置失败:', error);
          throw error;
        }
      },

      // 设置主题
      setTheme: (theme) => {
        get().updatePreferences({ theme });
      },

      // 设置分页大小
      setPageSize: (size) => {
        get().updatePreferences({ pageSize: size });
      },
    }),
    {
      name: 'settings-storage',
      partialize: (state) => ({
        preferences: state.preferences,
      }),
    }
  )
);
