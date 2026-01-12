/**
 * 主题提供者组件
 * 在应用根部包裹，提供主题功能
 */

import React, { useEffect, createContext, useContext, useRef } from 'react';
import { useThemeStore } from '@/store/themeStore';

interface ThemeContextValue {
  theme: string;
}

const ThemeContext = createContext<ThemeContextValue>({
  theme: 'minimal',
});

export const useThemeContext = () => useContext(ThemeContext);

export const ThemeProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { currentTheme, themeConfig } = useThemeStore();
  const previousThemeRef = useRef<string>(currentTheme);

  useEffect(() => {
    const root = document.documentElement;
    const previousTheme = previousThemeRef.current;

    // 如果主题发生变化，添加主题切换动画类
    if (previousTheme !== currentTheme) {
      root.classList.add('theme-switching');

      // 设置新主题
      root.setAttribute('data-theme', currentTheme);

      // 动画结束后移除类
      const timer = setTimeout(() => {
        root.classList.remove('theme-switching');
      }, 300);

      previousThemeRef.current = currentTheme;

      return () => clearTimeout(timer);
    } else {
      // 初始化时应用主题
      root.setAttribute('data-theme', currentTheme);
    }
  }, [currentTheme]);

  // 应用主题CSS变量
  useEffect(() => {
    const root = document.documentElement;

    // 应用颜色
    root.style.setProperty('--theme-bg-primary', themeConfig.colors.background.primary);
    root.style.setProperty('--theme-bg-secondary', themeConfig.colors.background.secondary);
    root.style.setProperty('--theme-bg-tertiary', themeConfig.colors.background.tertiary);
    root.style.setProperty('--theme-bg-surface', themeConfig.colors.background.surface);
    root.style.setProperty('--theme-bg-elevated', themeConfig.colors.background.elevated);

    root.style.setProperty('--theme-text-primary', themeConfig.colors.text.primary);
    root.style.setProperty('--theme-text-secondary', themeConfig.colors.text.secondary);
    root.style.setProperty('--theme-text-tertiary', themeConfig.colors.text.tertiary);
    root.style.setProperty('--theme-text-disabled', themeConfig.colors.text.disabled);
    root.style.setProperty('--theme-text-inverse', themeConfig.colors.text.inverse);

    root.style.setProperty('--theme-border-primary', themeConfig.colors.border.primary);
    root.style.setProperty('--theme-border-secondary', themeConfig.colors.border.secondary);
    root.style.setProperty('--theme-border-divider', themeConfig.colors.border.divider);

    root.style.setProperty('--theme-success', themeConfig.colors.success[1]);
    root.style.setProperty('--theme-warning', themeConfig.colors.warning[1]);
    root.style.setProperty('--theme-error', themeConfig.colors.error[1]);
    root.style.setProperty('--theme-info', themeConfig.colors.info[1]);

    // 应用间距
    root.style.setProperty('--theme-spacing-xs', themeConfig.spacing.xs);
    root.style.setProperty('--theme-spacing-sm', themeConfig.spacing.sm);
    root.style.setProperty('--theme-spacing-md', themeConfig.spacing.md);
    root.style.setProperty('--theme-spacing-lg', themeConfig.spacing.lg);
    root.style.setProperty('--theme-spacing-xl', themeConfig.spacing.xl);
    root.style.setProperty('--theme-spacing-2xl', themeConfig.spacing['2xl']);

    // 应用阴影
    root.style.setProperty('--theme-shadow-sm', themeConfig.shadows.sm);
    root.style.setProperty('--theme-shadow-md', themeConfig.shadows.md);
    root.style.setProperty('--theme-shadow-lg', themeConfig.shadows.lg);
    root.style.setProperty('--theme-shadow-xl', themeConfig.shadows.xl);

    // 应用圆角
    root.style.setProperty('--theme-radius-sm', themeConfig.radius.sm);
    root.style.setProperty('--theme-radius-md', themeConfig.radius.md);
    root.style.setProperty('--theme-radius-lg', themeConfig.radius.lg);
    root.style.setProperty('--theme-radius-xl', themeConfig.radius.xl);
    root.style.setProperty('--theme-radius-2xl', themeConfig.radius['2xl']);

    // 应用效果
    if (themeConfig.effects.glass) {
      root.style.setProperty('--theme-glass-blur', themeConfig.effects.glass.blur);
      root.style.setProperty('--theme-glass-opacity', themeConfig.effects.glass.opacity);
      root.style.setProperty('--theme-glass-border', themeConfig.effects.glass.border);
    }

    if (themeConfig.effects.neumorphic) {
      root.style.setProperty('--theme-neu-light', themeConfig.effects.neumorphic.light);
      root.style.setProperty('--theme-neu-dark', themeConfig.effects.neumorphic.dark);
      root.style.setProperty('--theme-neu-radius', themeConfig.effects.neumorphic.radius);
    }

    if (themeConfig.effects.transition) {
      root.style.setProperty('--theme-transition-fast', themeConfig.effects.transition.fast);
      root.style.setProperty('--theme-transition-base', themeConfig.effects.transition.base);
      root.style.setProperty('--theme-transition-slow', themeConfig.effects.transition.slow);
    }
  }, [themeConfig]);

  return (
    <ThemeContext.Provider value={{ theme: currentTheme }}>
      {children}
    </ThemeContext.Provider>
  );
};
