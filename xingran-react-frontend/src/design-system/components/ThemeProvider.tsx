/**
 * 主题提供者组件
 * 在应用根部包裹，提供主题功能
 */

import { useEffect, createContext, useContext, useRef } from "react";
import type { FC, ReactNode } from "react";
import { useThemeStore } from "@/store/themeStore";

interface ThemeContextValue {
  theme: string;
  mode: "light" | "dark";
}

const ThemeContext = createContext<ThemeContextValue>({
  theme: "minimal",
  mode: "light",
});

export const useThemeContext = () => useContext(ThemeContext);

const ThemeProvider: FC<{ children: ReactNode }> = ({ children }) => {
  const { appliedTheme, appliedMode } = useThemeStore();
  const previousThemeRef = useRef<string>(appliedTheme);

  useEffect(() => {
    const root = document.documentElement;
    const previousTheme = previousThemeRef.current;

    // 如果主题发生变化，添加主题切换动画类
    if (previousTheme !== appliedTheme) {
      root.classList.add("theme-switching");

      // 设置新主题
      root.setAttribute("data-theme", appliedTheme);
      root.setAttribute("data-color-mode", appliedMode);

      // 动画结束后移除类
      const timer = setTimeout(() => {
        root.classList.remove("theme-switching");
      }, 300);

      previousThemeRef.current = appliedTheme;

      return () => clearTimeout(timer);
    } else {
      // 初始化时应用主题
      root.setAttribute("data-theme", appliedTheme);
      root.setAttribute("data-color-mode", appliedMode);
    }
  }, [appliedTheme, appliedMode]);

  const contextValue: ThemeContextValue = {
    theme: appliedTheme,
    mode: appliedMode,
  };

  return <ThemeContext.Provider value={contextValue}>{children}</ThemeContext.Provider>;
};

export { ThemeProvider };
