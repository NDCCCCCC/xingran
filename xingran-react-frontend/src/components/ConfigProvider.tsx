/**
 * 配置初始化和同步 Provider
 * Configuration Provider
 *
 * 负责在应用启动时加载用户配置，并同步到各个 Store
 */

import { useEffect, type FC, type ReactNode } from "react";
import { useAuthStore } from "@/store/authStore";
import { useSettingsStore } from "@/store/settingsStore";
import { useThemeStore } from "@/store/themeStore";
import { useLayoutStore } from "@/store/layoutStore";

interface ConfigProviderProps {
  children: ReactNode;
}

/**
 * 配置 Provider 组件
 *
 * 功能：
 * 1. 用户登录后自动加载配置
 * 2. 将配置同步到各个 Store
 * 3. 处理配置的保存事件
 */
export const ConfigProvider: FC<ConfigProviderProps> = ({ children }) => {
  const { isAuthenticated } = useAuthStore();
  const { initialize, initialized, preferences } = useSettingsStore();

  // 同步方法
  const syncTheme = useThemeStore((state) => state.syncFromSettings);
  const syncLayout = useLayoutStore((state) => state.syncFromSettings);

  // 用户登录后初始化配置
  useEffect(() => {
    if (isAuthenticated && !initialized) {
      initialize();
    }
  }, [isAuthenticated, initialized, initialize]);

  // 监听配置变化，同步到各 Store
  useEffect(() => {
    if (!initialized) return;

    // 同步主题配置
    syncTheme(preferences.theme);

    // 同步布局配置
    syncLayout(preferences.layout);
  }, [preferences, initialized, syncTheme, syncLayout]);

  // 监听保存事件
  useEffect(() => {
    const handleSaveTheme = (event: CustomEvent) => {
      const { updateTheme } = useSettingsStore.getState();
      updateTheme(event.detail);
    };

    const handleSaveLayout = (event: CustomEvent) => {
      const { updateLayout } = useSettingsStore.getState();
      updateLayout(event.detail);
    };

    window.addEventListener("save-theme-settings", handleSaveTheme as EventListener);
    window.addEventListener("save-layout-settings", handleSaveLayout as EventListener);

    return () => {
      window.removeEventListener("save-theme-settings", handleSaveTheme as EventListener);
      window.removeEventListener("save-layout-settings", handleSaveLayout as EventListener);
    };
  }, []);

  return children;
};

export default ConfigProvider;
