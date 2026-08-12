/**
 * 菜单缓存接口定义
 * 用于短期内存缓存菜单和权限数据
 */

import type { Menu } from "@/types/system";

/**
 * 菜单缓存条目
 */
interface MenuCacheEntry<T> {
	data: T;
	timestamp: number;
	ttl: number;
}

/**
 * 菜单缓存接口
 */
export interface IMenuCache {
	/**
	 * 获取缓存的可见菜单树
	 */
	getMenus(): Menu[] | null;

	/**
	 * 获取所有菜单（包含隐藏菜单）
	 */
	getAllMenus(): Menu[] | null;

	/**
	 * 获取权限列表
	 */
	getPermissions(): string[] | null;

	/**
	 * 设置菜单缓存
	 * @param visibleMenus 可见菜单树
	 * @param allMenus 所有菜单
	 * @param permissions 权限列表
	 * @param ttl TTL（毫秒），默认 5 分钟
	 */
	setMenus(
		visibleMenus: Menu[],
		allMenus: Menu[],
		permissions: string[],
		ttl?: number
	): void;

	/**
	 * 检查缓存是否有效
	 */
	isValid(): boolean;

	/**
	 * 清空所有缓存
	 */
	clear(): void;

	/**
	 * 获取缓存剩余时间（秒）
	 */
	getRemainingTime(): number;
}

// 导出缓存条目类型供实现使用
export type { MenuCacheEntry };
