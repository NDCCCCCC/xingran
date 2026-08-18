/**
 * 主题提供者组件
 * 在应用根部包裹，提供明暗模式上下文
 *
 * v1.22 Phase 65（THEME-01）：多主题移除后仅订阅 themeStore 的 mode；
 * 不再写 data-theme 属性（全仓 CSS 无 [data-theme] 选择器），
 * 品牌色值由 index.css 静态定义，本组件只负责 data-color-mode 与切换动画。
 */

import { useEffect, createContext, useContext, useRef } from "react";
import type { FC, ReactNode } from "react";
import { useThemeStore } from "@/store/themeStore";
import type { ColorMode } from "@/types/config";

interface ThemeContextValue {
  mode: ColorMode;
}

const ThemeContext = createContext<ThemeContextValue>({
  mode: "light",
});

export const useThemeContext = () => useContext(ThemeContext);

const ThemeProvider: FC<{ children: ReactNode }> = ({ children }) => {
  const mode = useThemeStore((state) => state.mode);
  const previousModeRef = useRef<ColorMode>(mode);

  useEffect(() => {
    const root = document.documentElement;
    const previousMode = previousModeRef.current;

    // 如果模式发生变化，添加主题切换动画类
    if (previousMode !== mode) {
      root.classList.add("theme-switching");

      // 设置新模式
      root.setAttribute("data-color-mode", mode);

      // 动画结束后移除类
      const timer = setTimeout(() => {
        root.classList.remove("theme-switching");
      }, 300);

      previousModeRef.current = mode;

      return () => clearTimeout(timer);
    } else {
      // 初始化时应用模式
      root.setAttribute("data-color-mode", mode);
    }
  }, [mode]);

  const contextValue: ThemeContextValue = {
    mode,
  };

  return <ThemeContext.Provider value={contextValue}>{children}</ThemeContext.Provider>;
};

export { ThemeProvider };
