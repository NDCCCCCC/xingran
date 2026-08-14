/**
 * 系统设置状态管理（重构版）
 * System Settings State Management (Refactored)
 *
 * 这是所有用户配置的权威数据源（Single Source of Truth）
 * 其他 Store（themeStore, layoutStore）的配置都从这里衍生
 */

import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { UserPreferences, ThemeConfiguration, LayoutConfiguration } from "@/types/config";
import { defaultUserPreferences } from "@/types/config";
import { configService } from "@/services/configService";
import { ZUSTAND_STORAGE_KEYS } from "@/constants/storage";

interface SettingsState {
  // 权威数据
  preferences: UserPreferences;

  // 状态标识
  initialized: boolean;
  loading: boolean;
  error: string | null;

  // 版本管理
  version: number;
}

interface SettingsActions {
  // 初始化（从服务器加载）
  initialize: () => Promise<void>;

  // 更新配置
  updateTheme: (theme: Partial<ThemeConfiguration>) => Promise<void>;
  updateLayout: (layout: Partial<LayoutConfiguration>) => Promise<void>;
  updateDataPageSize: (pageSize: number) => Promise<void>;

  // 批量更新
  updatePreferences: (prefs: Partial<UserPreferences>) => Promise<void>;

  // 重置
  reset: () => void;

  // 导入/导出
  exportPreferences: () => string;
  importPreferences: (json: string) => void;

  // 内部方法
  syncToStores: () => void;
}

type SettingsStore = SettingsState & SettingsActions;

export const useSettingsStore = create<SettingsStore>()(
  persist(
    (set, get) => ({
      // 初始状态
      preferences: defaultUserPreferences,
      initialized: false,
      loading: false,
      error: null,
      version: 2,

      // 初始化 - 从服务器加载用户配置
      initialize: async () => {
        set({ loading: true, error: null });

        try {
          // 1. 从服务器获取配置
          const serverPrefs = await configService.getUserPreferences();

          // 2. 数据迁移（如果版本不匹配）
          const migratedPrefs = configService.migratePreferences(serverPrefs);

          // 3. 更新 Store
          set({
            preferences: migratedPrefs,
            initialized: true,
            loading: false,
          });

          // 4. 触发其他 Store 的同步（通过事件）
          get().syncToStores();
        } catch (error) {
          console.error("Failed to initialize settings:", error);
          set({
            loading: false,
            error: (error as Error).message,
            initialized: true, // 即使失败也标记为已初始化
          });
        }
      },

      // 更新主题配置
      updateTheme: async (themeUpdate) => {
        const { preferences } = get();
        const updatedPreferences: UserPreferences = {
          ...preferences,
          theme: {
            ...preferences.theme,
            ...themeUpdate,
          },
        };

        await get().updatePreferences(updatedPreferences);
      },

      // 更新布局配置
      updateLayout: async (layoutUpdate) => {
        const { preferences } = get();
        const updatedPreferences: UserPreferences = {
          ...preferences,
          layout: {
            ...preferences.layout,
            ...layoutUpdate,
          },
        };

        await get().updatePreferences(updatedPreferences);
      },

      // 更新分页大小
      updateDataPageSize: async (pageSize) => {
        const { preferences } = get();
        const updatedPreferences: UserPreferences = {
          ...preferences,
          data: {
            ...preferences.data,
            defaultPageSize: pageSize,
          },
        };

        await get().updatePreferences(updatedPreferences);
      },

      // 批量更新配置
      updatePreferences: async (updates) => {
        const { preferences } = get();
        const updatedPreferences = {
          ...preferences,
          ...updates,
        };

        try {
          // 1. 保存到服务器
          await configService.updateUserPreferences(updatedPreferences);

          // 2. 更新本地 Store
          set({ preferences: updatedPreferences });

          // 3. 同步到其他 Store
          get().syncToStores();
        } catch (error) {
          console.error("Failed to update preferences:", error);
          throw error;
        }
      },

      // 同步到其他 Store（内部方法）
      syncToStores: () => {
        const { preferences } = get();

        // 通过事件系统通知其他 Store
        // 这样可以避免循环依赖
        if (typeof window !== "undefined") {
          window.dispatchEvent(
            new CustomEvent("settings-changed", {
              detail: preferences,
            })
          );
        }
      },

      // 重置为默认值
      reset: () => {
        set({ preferences: defaultUserPreferences });
        get().syncToStores();
      },

      // 导出配置
      exportPreferences: () => {
        const { preferences } = get();
        return JSON.stringify(preferences, null, 2);
      },

      // 导入配置
      importPreferences: (json) => {
        try {
          const imported = JSON.parse(json);
          const migrated = configService.migratePreferences(imported);
          set({ preferences: migrated });
          get().syncToStores();
        } catch (error) {
          console.error("Failed to import preferences:", error);
          throw new Error("Invalid configuration file");
        }
      },
    }),
    {
      name: ZUSTAND_STORAGE_KEYS.SETTINGS,
      partialize: (state) => ({
        preferences: state.preferences,
        version: state.version,
      }),
    }
  )
);

// 类型选择器，用于避免不必要的重渲染
export const selectPreferences = (state: SettingsStore) => state.preferences;
export const selectThemeConfig = (state: SettingsStore) => state.preferences.theme;
export const selectLayoutConfig = (state: SettingsStore) => state.preferences.layout;
export const selectDataConfig = (state: SettingsStore) => state.preferences.data;
export const selectDefaultPageSize = (state: SettingsStore) =>
  state.preferences.data.defaultPageSize;
