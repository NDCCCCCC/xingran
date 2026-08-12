/**
 * 基于 TTL 的菜单内存缓存实现
 * 短期缓存菜单和权限数据，5 分钟自动过期
 */

import type { Menu } from "@/types/system";
import type { IMenuCache, MenuCacheEntry } from "./MenuCache";

/**
 * 默认 TTL：5 分钟
 */
const DEFAULT_TTL = 5 * 60 * 1000;

/**
 * 基于 Map 的 TTL 内存缓存实现
 */
export class TTLMenuCache implements IMenuCache {
	private menusCache: MenuCacheEntry<Menu[]> | null = null;
	private allMenusCache: MenuCacheEntry<Menu[]> | null = null;
	private permissionsCache: MenuCacheEntry<string[]> | null = null;
	private ttl: number;

	constructor(ttl: number = DEFAULT_TTL) {
		this.ttl = ttl;
	}

	getMenus(): Menu[] | null {
		if (!this.isEntryValid(this.menusCache)) {
			this.menusCache = null;
			return null;
		}
		return this.menusCache?.data ?? null;
	}

	getAllMenus(): Menu[] | null {
		if (!this.isEntryValid(this.allMenusCache)) {
			this.allMenusCache = null;
			return null;
		}
		return this.allMenusCache?.data ?? null;
	}

	getPermissions(): string[] | null {
		if (!this.isEntryValid(this.permissionsCache)) {
			this.permissionsCache = null;
			return null;
		}
		return this.permissionsCache?.data ?? null;
	}

	setMenus(
		visibleMenus: Menu[],
		allMenus: Menu[],
		permissions: string[],
		ttl?: number
	): void {
		const now = Date.now();
		const entryTtl = ttl ?? this.ttl;

		this.menusCache = { data: visibleMenus, timestamp: now, ttl: entryTtl };
		this.allMenusCache = { data: allMenus, timestamp: now, ttl: entryTtl };
		this.permissionsCache = { data: permissions, timestamp: now, ttl: entryTtl };
	}

	isValid(): boolean {
		return (
			this.isEntryValid(this.menusCache) &&
			this.isEntryValid(this.allMenusCache) &&
			this.isEntryValid(this.permissionsCache)
		);
	}

	clear(): void {
		this.menusCache = null;
		this.allMenusCache = null;
		this.permissionsCache = null;
	}

	getRemainingTime(): number {
		if (!this.menusCache) return 0;
		const elapsed = Date.now() - this.menusCache.timestamp;
		return Math.max(0, Math.floor((this.menusCache.ttl - elapsed) / 1000));
	}

	/**
	 * 检查缓存条目是否有效
	 */
	private isEntryValid<T>(entry: MenuCacheEntry<T> | null): boolean {
		if (!entry) return false;
		const elapsed = Date.now() - entry.timestamp;
		return elapsed < entry.ttl;
	}
}

/**
 * 单例模式的全局菜单缓存实例
 */
let globalMenuCacheInstance: TTLMenuCache | null = null;

/**
 * 获取菜单缓存实例
 * @param ttl 可选的 TTL 值（仅在首次创建时生效）
 */
export function getMenuCache(ttl?: number): TTLMenuCache {
	if (!globalMenuCacheInstance) {
		globalMenuCacheInstance = new TTLMenuCache(ttl);
	}
	return globalMenuCacheInstance;
}

/**
 * 重置菜单缓存（用于测试或特殊情况）
 */
export function resetMenuCache(): void {
	globalMenuCacheInstance = null;
}
