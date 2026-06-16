/**
 * 菜单状态管理（重构版 - TTL 内存缓存）
 * 使用 5 分钟 TTL 短期缓存菜单和权限数据
 *
 * 变更说明：
 * - 移除 localStorage 持久化
 * - 使用 TTLMenuCache 实现短期内存缓存（5 分钟）
 * - 页面刷新时从 API 重新获取
 * - 401 时自动清空缓存
 */

import { create } from "zustand";
import { getUserMenus, getAllUserMenus, getUserPermissions } from "@/lib/menuApi";
import type { Menu } from "@/types";
import { getMenuCache } from "@/services/cache/TTLMenuCache";
import { ZUSTAND_STORAGE_KEYS } from "@/constants/storage";

interface MenuState {
	menus: Menu[];
	allMenus: Menu[];
	permissions: string[];
	loading: boolean;
	lastFetchTime: number | null;
	error: Error | null;
}

interface MenuActions {
	fetchMenus: (forceRefresh?: boolean) => Promise<void>;
	fetchPermissions: (forceRefresh?: boolean) => Promise<void>;
	fetchAll: (forceRefresh?: boolean) => Promise<void>;
	setMenus: (menus: Menu[]) => void;
	setPermissions: (permissions: string[]) => void;
	clearMenus: () => void;
	invalidateCache: () => void;
	getCacheStatus: () => { isValid: boolean; remainingTime: number };
}

type MenuStore = MenuState & MenuActions;

/**
 * 合并菜单和权限获取
 */
const fetchMenuData = async () => {
	const [visibleMenus, allMenuList, permissions] = await Promise.all([
		getUserMenus(),
		getAllUserMenus(),
		getUserPermissions(),
	]);
	return { visibleMenus, allMenuList, permissions };
};

export const useMenuStore = create<MenuStore>()((set, get) => ({
	menus: [],
	allMenus: [],
	permissions: [],
	loading: false,
	lastFetchTime: null,
	error: null,

	fetchMenus: async (forceRefresh = false) => {
		const cache = getMenuCache();

		// 如果不强制刷新且缓存有效，直接使用缓存
		if (!forceRefresh) {
			const cachedMenus = cache.getMenus();
			const cachedAllMenus = cache.getAllMenus();
			if (cachedMenus && cachedAllMenus) {
				set({
					menus: cachedMenus,
					allMenus: cachedAllMenus,
					lastFetchTime: Date.now(),
					error: null
				});
				return;
			}
		}

		set({ loading: true, error: null });
		try {
			const { visibleMenus, allMenuList } = await fetchMenuData();

			// 更新缓存
			cache.setMenus(visibleMenus, allMenuList, get().permissions);

			set({
				menus: visibleMenus,
				allMenus: allMenuList,
				loading: false,
				lastFetchTime: Date.now(),
				error: null
			});
		} catch (error) {
			set({
				loading: false,
				error: error as Error,
				lastFetchTime: null
			});
			throw error;
		}
	},

	fetchPermissions: async (forceRefresh = false) => {
		const cache = getMenuCache();

		if (!forceRefresh) {
			const cachedPermissions = cache.getPermissions();
			if (cachedPermissions) {
				set({ permissions: cachedPermissions });
				return;
			}
		}

		try {
			const permissions = await getUserPermissions();

			// 更新缓存中的权限（保留现有菜单数据）
			const currentMenus = cache.getMenus() ?? [];
			const currentAllMenus = cache.getAllMenus() ?? [];
			cache.setMenus(currentMenus, currentAllMenus, permissions);

			set({ permissions, error: null });
		} catch (error) {
			set({ error: error as Error });
			throw error;
		}
	},

	fetchAll: async (forceRefresh = false) => {
		const cache = getMenuCache();

		if (!forceRefresh && cache.isValid()) {
			const cachedMenus = cache.getMenus();
			const cachedAllMenus = cache.getAllMenus();
			const cachedPermissions = cache.getPermissions();
			if (cachedMenus && cachedAllMenus && cachedPermissions) {
				set({
					menus: cachedMenus,
					allMenus: cachedAllMenus,
					permissions: cachedPermissions,
					lastFetchTime: Date.now(),
					error: null
				});
				return;
			}
		}

		set({ loading: true, error: null });
		try {
			const { visibleMenus, allMenuList, permissions } = await fetchMenuData();

			cache.setMenus(visibleMenus, allMenuList, permissions);

			set({
				menus: visibleMenus,
				allMenus: allMenuList,
				permissions,
				loading: false,
				lastFetchTime: Date.now(),
				error: null
			});
		} catch (error) {
			set({
				loading: false,
				error: error as Error,
				lastFetchTime: null
			});
			throw error;
		}
	},

	setMenus: (menus) => {
		const cache = getMenuCache();
		const allMenus = cache.getAllMenus() ?? [];
		const permissions = cache.getPermissions() ?? [];
		cache.setMenus(menus, allMenus, permissions);
		set({ menus });
	},

	setPermissions: (permissions) => {
		const cache = getMenuCache();
		const menus = cache.getMenus() ?? [];
		const allMenus = cache.getAllMenus() ?? [];
		cache.setMenus(menus, allMenus, permissions);
		set({ permissions });
	},

	clearMenus: () => {
		const cache = getMenuCache();
		cache.clear();
		set({
			menus: [],
			allMenus: [],
			permissions: [],
			lastFetchTime: null,
			error: null
		});
	},

	invalidateCache: () => {
		const cache = getMenuCache();
		cache.clear();
	},

	getCacheStatus: () => {
		const cache = getMenuCache();
		return {
			isValid: cache.isValid(),
			remainingTime: cache.getRemainingTime()
		};
	},
}));

/**
 * 刷新菜单缓存（强制从 API 获取）
 */
export const refreshMenuCache = async () => {
	await useMenuStore.getState().fetchAll(true);
};

/**
 * 清空菜单缓存
 */
export const clearMenuCache = () => {
	// 清除旧的 localStorage 数据（如果有）
	localStorage.removeItem(ZUSTAND_STORAGE_KEYS.MENU);
	useMenuStore.getState().clearMenus();
};
