/**
 * LAST_PATH (xingran_last_visited_path) sessionStorage 存取工具
 *
 * 从 DynamicRoutes.tsx 拆出（Phase 81 NAV-01）：
 *   1. react-refresh 清偿 —— 非组件导出与组件同文件触发 warning（原 :85/:95/:108）
 *   2. F5 防自毁语义归口 —— saveLastPath 的调用方守卫与这里集中管理
 */
import { STORAGE_KEYS } from "@/constants/storage";

// Get last visited path from sessionStorage
export const getLastPath = (): string | null => {
  if (typeof window === "undefined") return null;
  try {
    return sessionStorage.getItem(STORAGE_KEYS.LAST_PATH);
  } catch {
    return null;
  }
};

// Save current path to sessionStorage
export const saveLastPath = (path: string): void => {
  if (typeof window === "undefined") return;
  try {
    // 保存所有路径，除了 /login
    if (path && path !== "/login") {
      sessionStorage.setItem(STORAGE_KEYS.LAST_PATH, path);
    }
  } catch {
    // Ignore sessionStorage errors
  }
};

// Clear last path (for logout)
export const clearLastPath = (): void => {
  if (typeof window === "undefined") return;
  try {
    sessionStorage.removeItem(STORAGE_KEYS.LAST_PATH);
  } catch {
    // Ignore sessionStorage errors
  }
};
