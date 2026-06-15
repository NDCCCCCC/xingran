/**
 * Storage Key 常量
 *
 * 集中管理前端 localStorage / sessionStorage / Zustand persist 使用的 key 字符串，
 * 避免在多个文件中硬编码同一字符串导致 key 漂移或重命名遗漏。
 *
 * 注意：
 * - token 存储相关 key（rt / tm）保留在 SecureTokenStorageImpl.ts 内部，
 *   因为它们是加密 token 专用的简短 key，外部不应直接引用。
 * - 路由 last visited path（ruoyi_last_visited_path）目前保留在 DynamicRoutes.tsx
 *   本地，未来可统一迁移至此文件。
 */

/**
 * Zustand persist storage keys
 * 与 F-STORE-02 修复配套使用，集中 3 个 store 的 storage key
 */
export const ZUSTAND_STORAGE_KEYS = {
	SETTINGS: 'settings-storage',
	LAYOUT: 'layout-storage',
	MENU: 'menu-storage',
} as const;

/**
 * 类型导出：所有 Zustand persist key 的联合类型
 * 用于需要运行时校验 key 合法性的场景
 */
export type ZustandStorageKey =
	(typeof ZUSTAND_STORAGE_KEYS)[keyof typeof ZUSTAND_STORAGE_KEYS];
