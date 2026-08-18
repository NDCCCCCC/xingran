/**
 * 主题状态管理（v1.22 Phase 65 · THEME-01/THEME-02 收敛版）
 * Theme State Management (Single Color-Mode Store)
 *
 * 多主题能力已随 D-01 移除：本 store 仅负责 light / dark 明暗模式，
 * 并把当前模式写入 document.documentElement[data-color-mode]。
 * 品牌色值由 src/index.css 静态定义（唯一来源 brand-spec.md，per D-02），
 * 不再有任何运行时主题变量注入。
 */

import { create } from "zustand";
import type { ColorMode } from "@/types/config";

interface ThemeState {
  /** 当前明暗模式（从 SettingsStore 衍生） */
  mode: ColorMode;
}

interface ThemeActions {
  /** 从 SettingsStore 同步明暗模式 */
  syncFromSettings: (config: { mode: ColorMode }) => void;

  /** 直接设置明暗模式 */
  setMode: (mode: ColorMode) => void;

  /** 应用当前模式到 DOM（写 data-color-mode 属性） */
  applyToDOM: () => void;
}

type ThemeStore = ThemeState & ThemeActions;

export const useThemeStore = create<ThemeStore>()((set, get) => ({
  mode: "light",

  // 从 SettingsStore 同步配置
  syncFromSettings: (config) => {
    set({ mode: config.mode });
    get().applyToDOM();
  },

  // 直接设置模式
  setMode: (mode) => {
    set({ mode });
    get().applyToDOM();
  },

  // 应用到 DOM：只做一件事 —— 写 data-color-mode
  // （品牌色值由 index.css :root / [data-color-mode] 静态提供）
  applyToDOM: () => {
    const { mode } = get();
    document.documentElement.setAttribute("data-color-mode", mode);
  },
}));

// 监听 SettingsStore 变化
// P1-M3: 命名函数 + 先移除再注册,防止 Vite HMR 重复注册导致回调触发 N 次
function handleSettingsChangedForTheme(event: Event) {
  const preferences = (event as CustomEvent).detail;
  const syncTheme = useThemeStore.getState().syncFromSettings;
  syncTheme({ mode: preferences.theme.mode });
}
if (typeof window !== "undefined") {
  window.removeEventListener("settings-changed", handleSettingsChangedForTheme);
  window.addEventListener("settings-changed", handleSettingsChangedForTheme);
}
