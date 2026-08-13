/**
 * useWidgetPolling - Widget 数据轮询 Hook
 *
 * 实现 Widget 数据定时刷新，支持：
 * - 可配置刷新间隔
 * - 缓存检查（避免重复请求）
 * - Page Visibility API 优化（页面不可见时暂停）
 * - 手动刷新
 *
 * 修复历史:
 *   - P0-2/P0-3 (前端审查): 原实现主 effect 与 visibility effect 共用同一个
 *     intervalRef, 互相覆盖句柄导致 interval 永远 clear 不掉(泄漏); 且
 *     widgetIds 数组引用不稳定使 fetchData 频繁重建触发 effect 死循环。
 *     现统一为单一 effect 管理 interval, visibility 通过 isTabVisible 状态
 *     驱动, fetcher 用 ref 保持最新闭包避免依赖抖动。
 */

import { useState, useEffect, useRef, useCallback } from "react";
import { useDashboardStore } from "@/store/dashboardStore";
import { dashboardService } from "@/services/dashboardService";

export interface UseWidgetPollingOptions {
	/** 需要轮询的 Widget ID 列表 */
	widgetIds: string[];
	/** 刷新间隔（秒） */
	interval: number;
	/** 是否启用轮询 */
	enabled?: boolean;
	/** 最小缓存时间（秒），默认 30 秒 */
	minCacheTime?: number;
}

export interface UseWidgetPollingReturn {
	/** 是否正在加载 */
	loading: boolean;
	/** 最后刷新时间 */
	lastRefreshTime: Date | null;
	/** 手动刷新 */
	refresh: () => Promise<void>;
	/** 暂停轮询 */
	pause: () => void;
	/** 恢复轮询 */
	resume: () => void;
	/** 是否已暂停 */
	isPaused: boolean;
}

/**
 * Widget 数据轮询 Hook
 */
export function useWidgetPolling(options: UseWidgetPollingOptions): UseWidgetPollingReturn {
	const {
		widgetIds,
		interval,
		enabled = true,
		minCacheTime = 30,
	} = options;

	const { cacheWidgetData, getCachedWidgetData, clearWidgetCache } = useDashboardStore();

	const [loading, setLoading] = useState(false);
	const [lastRefreshTime, setLastRefreshTime] = useState<Date | null>(null);
	const [isPaused, setIsPaused] = useState(false);
	const [isTabVisible, setIsTabVisible] = useState(
		typeof document !== "undefined" ? !document.hidden : true
	);

	const isFetchingRef = useRef(false);

	// 计算缓存过期时间（毫秒）
	const cacheExpiry = Math.max((interval / 2) * 1000, minCacheTime * 1000);

	// 用 ref 保存最新的 widgetIds / cacheExpiry / store actions,避免它们进入
	// useCallback/fetchData 的依赖数组造成频繁重建(原 P0-3 根因)。
	// widgetIds 是数组,调用方很可能每次渲染传新引用,放 ref 后 fetcher 只需建一次。
	const widgetIdsRef = useRef(widgetIds);
	widgetIdsRef.current = widgetIds;
	const cacheExpiryRef = useRef(cacheExpiry);
	cacheExpiryRef.current = cacheExpiry;

	// 获取数据 — 空依赖,读 ref 拿最新值,保证引用永久稳定
	const fetchData = useCallback(async (forceRefresh = false) => {
		const ids = widgetIdsRef.current;
		if (ids.length === 0 || isFetchingRef.current) return;

		const expiry = cacheExpiryRef.current;
		const now = Date.now();
		const uncachedIds: string[] = [];

		for (const id of ids) {
			if (forceRefresh) {
				clearWidgetCache(id);
				uncachedIds.push(id);
			} else {
				const cached = getCachedWidgetData(id);
				if (!cached || typeof cached !== "object" || !("timestamp" in cached) || (now - (cached as { timestamp: number }).timestamp) > expiry) {
					uncachedIds.push(id);
				}
			}
		}

		if (uncachedIds.length === 0) return;

		isFetchingRef.current = true;
		setLoading(true);

		try {
			const data = await dashboardService.getBatchWidgetData(uncachedIds);
			for (const [id, widgetData] of data) {
				cacheWidgetData(id, widgetData);
			}
			setLastRefreshTime(new Date());
		} catch (error) {
			console.error("Failed to fetch widget data:", error);
		} finally {
			setLoading(false);
			isFetchingRef.current = false;
		}
	}, [cacheWidgetData, getCachedWidgetData, clearWidgetCache]);

	// 手动刷新
	const refresh = useCallback(async () => {
		await fetchData(true);
	}, [fetchData]);

	// 暂停轮询
	const pause = useCallback(() => {
		setIsPaused(true);
	}, []);

	// 恢复轮询
	const resume = useCallback(() => {
		setIsPaused(false);
	}, []);

	// 监听页面可见性 — 仅更新状态,不直接操作 interval(原 P0-2 根因:
	// 两个 effect 争抢 intervalRef)。interval 的创建/销毁全部交给下面的主 effect。
	useEffect(() => {
		const handleVisibilityChange = () => {
			setIsTabVisible(!document.hidden);
		};
		document.addEventListener("visibilitychange", handleVisibilityChange);
		return () => {
			document.removeEventListener("visibilitychange", handleVisibilityChange);
		};
	}, []);

	// 单一的 interval 管理 effect — 唯一负责 setInterval/clearInterval 的地方。
	// 当 enabled / isPaused / isTabVisible / interval 变化时重建,保证只有
	// 一个活跃 interval,不会泄漏。
	useEffect(() => {
		if (!enabled || isPaused || !isTabVisible || widgetIds.length === 0) return;

		fetchData();

		const intervalMs = Math.max(interval, 30) * 1000; // 最小 30 秒
		const id = setInterval(() => {
			fetchData();
		}, intervalMs);

		return () => {
			clearInterval(id);
		};
		// widgetIds.length 作为基本类型依赖(避免数组引用抖动); widgetIds 内容
		// 变化由 fetcher 内部 widgetIdsRef 捕获,无需进依赖。
	}, [enabled, isPaused, isTabVisible, interval, widgetIds.length, fetchData]);

	return {
		loading,
		lastRefreshTime,
		refresh,
		pause,
		resume,
		isPaused,
	};
}

export default useWidgetPolling;
