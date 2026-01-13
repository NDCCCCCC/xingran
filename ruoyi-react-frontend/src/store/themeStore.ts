/**
 * 主题状态管理
 * 使用Zustand进行主题切换和持久化
 */

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { ThemeType, ThemeConfig } from '@/types/theme';
import { themes, applyThemeVariables, applyEffectVariables } from '@/design-system/themes';

interface ThemeState {
  currentTheme: ThemeType;
  themeConfig: ThemeConfig;
}

interface ThemeActions {
  setTheme: (theme: ThemeType) => void;
  toggleTheme: () => void;
  resetTheme: () => void;
}

type ThemeStore = ThemeState & ThemeActions;

export const useThemeStore = create<ThemeStore>()(
  persist(
    (set, get) => ({
      // 初始状态
      currentTheme: 'minimal',
      themeConfig: themes.minimal,

      // 设置主题
      setTheme: (theme: ThemeType) => {
        const newThemeConfig = themes[theme];

        set({
          currentTheme: theme,
          themeConfig: newThemeConfig,
        });

        // 应用主题到DOM
        applyThemeVariables(newThemeConfig);
        applyEffectVariables(newThemeConfig.effects);

        // 设置data-theme属性
        document.documentElement.setAttribute('data-theme', theme);
      },

      // 切换到下一个主题
      toggleTheme: () => {
        const themeTypes: ThemeType[] = ['minimal', 'glassmorphism', 'neumorphism', 'flat2.0'];
        const currentIndex = themeTypes.indexOf(get().currentTheme);
        const nextTheme = themeTypes[(currentIndex + 1) % themeTypes.length];
        get().setTheme(nextTheme);
      },

      // 重置为默认主题
      resetTheme: () => {
        get().setTheme('minimal');
      },
    }),
    {
      name: 'theme-storage',
      // 只持久化主题类型，配置每次从themes中获取
      partialize: (state) => ({
        currentTheme: state.currentTheme,
      }),
      // 初始化时应用主题
      onRehydrateStorage: () => (state) => {
        if (state) {
          const themeConfig = themes[state.currentTheme];
          applyThemeVariables(themeConfig);
          applyEffectVariables(themeConfig.effects);
          document.documentElement.setAttribute('data-theme', state.currentTheme);
        }
      },
    }
  )
);

/**
 * 主题Hook
 * 提供更方便的主题访问和操作
 */
export function useTheme() {
  const { currentTheme, themeConfig, setTheme, toggleTheme } = useThemeStore();

  return {
    theme: currentTheme,
    config: themeConfig,
    setTheme,
    toggleTheme,
    isMinimal: currentTheme === 'minimal',
    isGlassmorphism: currentTheme === 'glassmorphism',
    isNeumorphism: currentTheme === 'neumorphism',
    isFlat2: currentTheme === 'flat2.0',
  };
}
