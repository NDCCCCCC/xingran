/**
 * useWidgetData - Widget数据Hook
 *
 * 获取和管理Widget数据
 */

import { useState, useEffect, useCallback, useRef, useMemo } from "react";
import { useDashboardStore } from "@/store/dashboardStore";
import { dataFetcher } from "@/components/dashboard/utils/dataFetcher";
import type { WidgetConfig } from "@/types/dashboard";

interface UseWidgetDataOptions {
	/** 是否禁用自动刷新 */
	disabled?: boolean;

	/** 刷新间隔（秒），默认使用Widget配置 */
	refreshInterval?: number;
}

interface UseWidgetDataResult<T = unknown> {
	/** 数据 */
	data: T | null;

	/** 加载状态 */
	loading: boolean;

	/** 错误信息 */
	error: string | null;

	/** 刷新数据 */
	refresh: () => Promise<void>;

	/** 是否正在刷新 */
	isRefreshing: boolean;
}

/**
 * Widget数据Hook
 */
export function useWidgetData<T = unknown>(
	widget: WidgetConfig,
	options?: UseWidgetDataOptions
): UseWidgetDataResult<T> {
	const { getCachedWidgetData, cacheWidgetData } = useDashboardStore();
	const [data, setData] = useState<T | null>(null);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [isRefreshing, setIsRefreshing] = useState(false);

	// 使用ref存储最新的值，避免闭包陷阱
	const widgetRef = useRef(widget);
	const optionsRef = useRef(options ?? {});
	const disabledRef = useRef(options?.disabled ?? false);

	// 使用useEffect来更新ref，而不是在render中
	useEffect(() => {
		widgetRef.current = widget;
		optionsRef.current = options ?? {};
		disabledRef.current = options?.disabled ?? false;
	});

	// 获取刷新间隔 - 稳定的引用
	const refreshInterval = useMemo(() => {
		return options?.refreshInterval ?? widget.refreshInterval ?? 60;
	}, [widget.refreshInterval, options?.refreshInterval]);

	// 获取数据
	const fetchData = useCallback(async (showLoading = true) => {
		const currentWidget = widgetRef.current;
		const isDisabled = disabledRef.current;

		if (isDisabled || !currentWidget.enabled) {
			return;
		}

		try {
			if (showLoading) {
				setLoading(true);
			} else {
				setIsRefreshing(true);
			}
			setError(null);

			// 尝试从缓存获取
			const cached = getCachedWidgetData(currentWidget.id);
			if (cached && !showLoading) {
				setData(cached as T);
				setIsRefreshing(false);
				return;
			}

			// 从数据源获取
			const result = await dataFetcher.fetch<T>(currentWidget.dataSource);

			if (result.error) {
				setError(result.error);
			} else {
				setData(result.data);
				// 缓存数据
				cacheWidgetData(currentWidget.id, result.data);
			}
		} catch (err) {
			setError((err as Error).message);
		} finally {
			setLoading(false);
			setIsRefreshing(false);
		}
	}, [getCachedWidgetData, cacheWidgetData]);

	// 手动刷新
	const refresh = useCallback(async () => {
		await fetchData(false);
	}, [fetchData]);

	// 初始加载和自动刷新
	useEffect(() => {
		if (options?.disabled || !widget.enabled) return;

		// 初始加载
		fetchData(true);

		// 设置自动刷新
		const interval = refreshInterval * 1000;
		if (interval > 0) {
			const timer = setInterval(() => {
				fetchData(false);
			}, interval);

			return () => clearInterval(timer);
		}
	}, [widget.enabled, widget.id, options?.disabled, refreshInterval, fetchData]);

	return {
		data,
		loading,
		error,
		refresh,
		isRefreshing,
	};
}

/**
 * 批量获取多个Widget数据的Hook
 */
export function useBatchWidgetData(
	widgets: WidgetConfig[],
	options?: UseWidgetDataOptions
): Record<string, unknown> {
	const [dataMap, setDataMap] = useState<Record<string, unknown>>({});
	const [loading, setLoading] = useState(true);

	useEffect(() => {
		if (options?.disabled) return;

		const fetchAll = async () => {
			setLoading(true);
			const results: Record<string, unknown> = {};

			await Promise.all(
				widgets.map(async (widget) => {
					try {
						const result = await dataFetcher.fetch(widget.dataSource);
						results[widget.id] = result.data;
					} catch (error) {
						results[widget.id] = null;
					}
				})
			);

			setDataMap(results);
			setLoading(false);
		};

		fetchAll();

		// 自动刷新
		const interval = widgets.reduce((min, w) => {
			const wi = w.refreshInterval ?? 60;
			return wi < min ? wi : min;
		}, 60) * 1000;

		if (interval > 0) {
			const timer = setInterval(fetchAll, interval);
			return () => clearInterval(timer);
		}
	}, [widgets, options?.disabled]);

	return { dataMap, loading };
}
