/**
 * useWidgetPolling - Widget 数据轮询 Hook
 *
 * 实现 Widget 数据定时刷新，支持：
 * - 可配置刷新间隔
 * - 缓存检查（避免重复请求）
 * - Page Visibility API 优化（页面不可见时暂停）
 * - 手动刷新
 */

import { useState, useEffect, useRef, useCallback } from "react";
import { useDashboardStore } from "@/store/dashboardStore";
import { dashboardService } from "@/services/dashboardService";
import type { WidgetConfig } from "@/types/dashboard";

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

	const { cacheWidgetData, getCachedWidgetData, clearWidgetCache, currentDashboard } = useDashboardStore();

	const [loading, setLoading] = useState(false);
	const [lastRefreshTime, setLastRefreshTime] = useState<Date | null>(null);
	const [isPaused, setIsPaused] = useState(false);

	const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
	const isFetchingRef = useRef(false);

	// 计算缓存过期时间（毫秒）
	const cacheExpiry = Math.max((interval / 2) * 1000, minCacheTime * 1000);

	// 获取数据
	const fetchData = useCallback(async (forceRefresh = false) => {
		if (widgetIds.length === 0 || isFetchingRef.current) return;

		// 检查缓存，确定需要请求的 Widget
		const now = Date.now();
		const uncachedIds: string[] = [];

		for (const id of widgetIds) {
			if (forceRefresh) {
				// 强制刷新时清除缓存
				clearWidgetCache(id);
				uncachedIds.push(id);
			} else {
				// 检查缓存是否过期
				const cached = getCachedWidgetData(id);
				if (!cached || typeof cached !== "object" || !("timestamp" in cached) || (now - (cached as { timestamp: number }).timestamp) > cacheExpiry) {
					uncachedIds.push(id);
				}
			}
		}

		// 如果所有数据都在缓存中且未过期，跳过请求
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
	}, [widgetIds, cacheExpiry, cacheWidgetData, getCachedWidgetData, clearWidgetCache]);

	// 手动刷新
	const refresh = useCallback(async () => {
		await fetchData(true);
	}, [fetchData]);

	// 暂停轮询
	const pause = useCallback(() => {
		setIsPaused(true);
		if (intervalRef.current) {
			clearInterval(intervalRef.current);
			intervalRef.current = null;
		}
	}, []);

	// 恢复轮询
	const resume = useCallback(() => {
		setIsPaused(false);
	}, []);

	// 设置轮询
	useEffect(() => {
		if (!enabled || isPaused || widgetIds.length === 0) return;

		// 首次加载
		fetchData();

		// 设置定时轮询
		const intervalMs = Math.max(interval, 30) * 1000; // 最小 30 秒
		intervalRef.current = setInterval(() => {
			fetchData();
		}, intervalMs);

		return () => {
			if (intervalRef.current) {
				clearInterval(intervalRef.current);
				intervalRef.current = null;
			}
		};
	}, [enabled, isPaused, widgetIds, interval, fetchData]);

	// Page Visibility API 优化
	useEffect(() => {
		const handleVisibilityChange = () => {
			if (document.hidden) {
				// 页面不可见时暂停轮询
				if (intervalRef.current) {
					clearInterval(intervalRef.current);
					intervalRef.current = null;
				}
			} else {
				// 页面可见时恢复轮询
				if (enabled && !isPaused && widgetIds.length > 0) {
					fetchData();
					const intervalMs = Math.max(interval, 30) * 1000;
					intervalRef.current = setInterval(() => {
						fetchData();
					}, intervalMs);
				}
			}
		};

		document.addEventListener("visibilitychange", handleVisibilityChange);

		return () => {
			document.removeEventListener("visibilitychange", handleVisibilityChange);
		};
	}, [enabled, isPaused, widgetIds, interval, fetchData]);

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
