/**
 * 用户设置页面（Phase 70 · D-05/D-06 行式重构版）
 * User Settings Page (SettingsShell instance)
 *
 * 分类注册表与设置卡片自 v1.24 P82 拆至 ./categories
 * （react-refresh 要求组件文件只导出组件）。
 */

import { useEffect, type FC } from "react";
import { SettingsShell } from "@/design-system/components/SettingsShell";
import { useSettingsStore } from "@/store/settingsStore";
import { userSettingsCategories } from "./categories";

// ---------- 页面入口 ----------

const SettingsPage: FC = () => {
  const initialized = useSettingsStore((s) => s.initialized);

  // 懒加载初始化（依赖为原始布尔值，稳定）
  useEffect(() => {
    if (!initialized) {
      useSettingsStore.getState().initialize();
    }
  }, [initialized]);

  if (!initialized) {
    return <div>加载中...</div>;
  }

  return <SettingsShell categories={userSettingsCategories} defaultCat="appearance" />;
};

export default SettingsPage;
