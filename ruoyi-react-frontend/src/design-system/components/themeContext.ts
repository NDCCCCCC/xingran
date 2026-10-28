/**
 * 主题 Context（v1.24 P82 自 ThemeProvider.tsx 拆出：
 * react-refresh 要求组件文件只导出组件，hook 与组件同文件会破坏 Fast Refresh 边界）
 */
import { createContext, useContext } from "react";
import type { ColorMode } from "@/types/config";

export interface ThemeContextValue {
  mode: ColorMode;
}

export const ThemeContext = createContext<ThemeContextValue>({
  mode: "light",
});

export const useThemeContext = () => useContext(ThemeContext);
