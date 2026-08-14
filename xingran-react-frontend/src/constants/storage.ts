/**
 * Storage Key 常量
 *
 * 集中管理前端 localStorage / sessionStorage / Zustand persist 使用的 key 字符串，
 * 避免在多个文件中硬编码同一字符串导致 key 漂移或重命名遗漏。
 *
 * 注意：
 * - token 存储相关 key（rt / tm）保留在 SecureTokenStorageImpl.ts 内部，
 *   因为它们是加密 token 专用的简短 key，外部不应直接引用。
 */

/**
 * Zustand persist storage keys
 * 与 F-STORE-02 修复配套使用，集中 3 个 store 的 storage key
 */
export const ZUSTAND_STORAGE_KEYS = {
  SETTINGS: "settings-storage",
  LAYOUT: "layout-storage",
  MENU: "menu-storage",
} as const;

/**
 * 类型导出：所有 Zustand persist key 的联合类型
 * 用于需要运行时校验 key 合法性的场景
 */
export type ZustandStorageKey = (typeof ZUSTAND_STORAGE_KEYS)[keyof typeof ZUSTAND_STORAGE_KEYS];

/**
 * Session / Local Storage keys
 * 用于非 Zustand persist 的浏览器原生存储 key 集中管理
 */
export const STORAGE_KEYS = {
  /**
   * 路由最后访问路径
   * 存储于 sessionStorage，用于登录后跳转到用户上次访问的页面
   */
  LAST_PATH: "xingran_last_visited_path",
} as const;

/**
 * 类型导出：所有原生 storage key 的联合类型
 */
export type StorageKey = (typeof STORAGE_KEYS)[keyof typeof STORAGE_KEYS];

/**
 * Table 状态持久化前缀（用于 useTableManager / usePagination / usePersistedState）
 * 与前缀名规则保持一致: `xingran_table_state_<sanitizedPath>_<slotName>`
 */
export const TABLE_STATE_PREFIX = "xingran_table_state_";

/**
 * 将 URL 路径转换为可作为 storage key 的一部分的字符串。
 * 规则:去除前导斜杠，将中间的所有 `/` 替换为 `_`，以便安全拼接到 key 中。
 *
 * @example
 * sanitizePathForKey("/system/user")  // "system_user"
 * sanitizePathForKey("/ops/buildings") // "ops_buildings"
 */
export function sanitizePathForKey(path: string): string {
  if (!path) return "";
  return path.replace(/^\/+/, "").replace(/\//g, "_");
}

/**
 * 清理指定路由路径对应的所有表格状态（filters/分页/排序）。
 * 按路径前缀精确匹配 sessionStorage key，仅删除该路径下的状态。
 *
 * 用于关闭应用内标签页时清理，满足"关闭标签页才取消状态"需求；
 * 也在切换部门/数据源等场景按需调用。
 *
 * 安全：仅删除 `TABLE_STATE_PREFIX` 命名空间下的查询参数态数据，
 * 不触及 token / 业务数据行 / 凭据。
 */
export function clearTableStateByPath(path: string): void {
  if (typeof window === "undefined") return;
  const sanitized = sanitizePathForKey(path);
  if (!sanitized) return;
  const prefix = `${TABLE_STATE_PREFIX}${sanitized}_`;
  try {
    const keysToRemove: string[] = [];
    for (let i = 0; i < sessionStorage.length; i++) {
      const key = sessionStorage.key(i);
      if (key && key.startsWith(prefix)) keysToRemove.push(key);
    }
    keysToRemove.forEach((k) => sessionStorage.removeItem(k));
  } catch {
    // 隐私模式 / 配额溢出 — 静默吞掉，不阻塞调用方
  }
}

/**
 * 清理全部表格状态（所有路径）。用于登出，确保换人登录无上一用户筛选/分页/排序痕迹。
 * 对标等保 2.0「剩余信息保护」最小实践。
 */
export function clearAllTableState(): void {
  if (typeof window === "undefined") return;
  try {
    const keysToRemove: string[] = [];
    for (let i = 0; i < sessionStorage.length; i++) {
      const key = sessionStorage.key(i);
      if (key && key.startsWith(TABLE_STATE_PREFIX)) keysToRemove.push(key);
    }
    keysToRemove.forEach((k) => sessionStorage.removeItem(k));
  } catch {
    // ignore
  }
}
