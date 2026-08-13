/**
 * useRealtimeUpdates - 实时更新Hook
 *
 * 通过WebSocket实现实时数据更新
 *
 * 修复历史:
 *   - P0-3 (前端审查): connect 依赖整个 options 对象,父组件内联 options 每次
 *     渲染都触发 WebSocket 断开重连,配合 5s 自动重连形成连接风暴。
 *   - P1-H2 (前端审查): onclose 读闭包旧 options?.enabled,enabled 切 false
 *     后仍重连; disconnect 与异步 onclose 竞态。
 *     现将 options/widgets 移入 ref,connect 读 ref,引用稳定。
 */

import { useEffect, useCallback, useRef } from "react";
import { useDashboardStore } from "@/store/dashboardStore";
import type { WidgetConfig, WebSocketDataSourceConfig, DataSourceConfig } from "@/types/dashboard";

// 类型守卫：检查 dataSource 是否有直接的 type 属性
function hasDataSourceType(ds: DataSourceConfig): ds is WebSocketDataSourceConfig {
	return "type" in ds && (ds as { type: string }).type === "websocket";
}

// 类型守卫：检查是否为 WebSocket 数据源
function isWebSocketDataSource(ds: DataSourceConfig): ds is WebSocketDataSourceConfig {
	if ("type" in ds) {
		const typed = ds as { type: string };
		return typed.type === "websocket";
	}
	// 检查包装格式 { websocket: WebSocketDataSourceConfig }
	if ("websocket" in ds) {
		return true;
	}
	return false;
}

interface UseRealtimeUpdatesOptions {
	/** 是否启用 */
	enabled?: boolean;

	/** 消息回调 */
	onMessage?: (widgetId: string, data: unknown) => void;

	/** 错误回调 */
	onError?: (error: Event) => void;

	/** 连接状态回调 */
	onConnectionChange?: (connected: boolean) => void;
}

/**
 * 实时更新Hook
 */
export function useRealtimeUpdates(
	widgets: WidgetConfig[],
	options?: UseRealtimeUpdatesOptions
) {
	const { cacheWidgetData } = useDashboardStore();
	const wsRef = useRef<WebSocket | null>(null);
	const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
	const subscribedChannelsRef = useRef<Set<string>>(new Set());
	// P1-H2: 标记是否已主动断开,防止 onclose 里的自动重连
	const isManualDisconnectRef = useRef(false);

	// P0-3: 用 ref 保存最新的 widgets / options,避免它们进 connect 依赖数组
	const widgetsRef = useRef(widgets);
	widgetsRef.current = widgets;
	const optionsRef = useRef(options);
	optionsRef.current = options;

	// 获取WebSocket URL
	const getWsUrl = useCallback(() => {
		const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
		const host = import.meta.env.VITE_WS_HOST || window.location.host;
		const basePath = import.meta.env.VITE_WS_BASE_PATH || "/ws";
		return `${protocol}//${host}${basePath}/dashboard`;
	}, []);

	// 连接WebSocket — 读 ref,自身引用稳定(仅依赖 getWsUrl)
	const connect = useCallback(() => {
		if (wsRef.current?.readyState === WebSocket.OPEN) {
			return;
		}

		isManualDisconnectRef.current = false;

		try {
			const ws = new WebSocket(getWsUrl());
			const opts = optionsRef.current;

			ws.onopen = () => {
				opts?.onConnectionChange?.(true);

				// 订阅所有WebSocket数据源的Widget
				widgetsRef.current.forEach((widget) => {
					if (isWebSocketDataSource(widget.dataSource)) {
						const channel = widget.dataSource.channel;
						if (!subscribedChannelsRef.current.has(channel)) {
							ws.send(JSON.stringify({
								action: "subscribe",
								channel,
							}));
							subscribedChannelsRef.current.add(channel);
						}
					}
				});
			};

			ws.onmessage = (event) => {
				try {
					const message = JSON.parse(event.data);

					if (message.type === "data_update") {
						const { widgetId, data } = message;
						// 更新缓存
						cacheWidgetData(widgetId, data);
						// 触发回调
						optionsRef.current?.onMessage?.(widgetId, data);
					}
				} catch (error) {
					console.error("[Dashboard WebSocket] Message parse error:", error);
				}
			};

			ws.onerror = (error) => {
				console.error("[Dashboard WebSocket] Error:", error);
				optionsRef.current?.onError?.(error);
				optionsRef.current?.onConnectionChange?.(false);
			};

			ws.onclose = () => {
				optionsRef.current?.onConnectionChange?.(false);
				subscribedChannelsRef.current.clear();

				// P1-H2: 主动断开时不重连; 读 ref 拿最新 enabled
				if (!isManualDisconnectRef.current && optionsRef.current?.enabled !== false) {
					reconnectTimerRef.current = setTimeout(() => {
						connect();
					}, 5000);
				}
			};

			wsRef.current = ws;
		} catch (error) {
			console.error("[Dashboard WebSocket] Connection error:", error);
		}
	}, [getWsUrl, cacheWidgetData]);

	// 断开连接
	const disconnect = useCallback(() => {
		isManualDisconnectRef.current = true;

		if (reconnectTimerRef.current) {
			clearTimeout(reconnectTimerRef.current);
			reconnectTimerRef.current = null;
		}

		if (wsRef.current) {
			wsRef.current.close();
			wsRef.current = null;
		}

		subscribedChannelsRef.current.clear();
	}, []);

	// 手动刷新订阅 — 读 ref
	const refreshSubscriptions = useCallback(() => {
		if (wsRef.current?.readyState === WebSocket.OPEN) {
			widgetsRef.current.forEach((widget) => {
				if (isWebSocketDataSource(widget.dataSource)) {
					const channel = widget.dataSource.channel;
					wsRef.current?.send(JSON.stringify({
						action: "subscribe",
						channel,
					}));
					subscribedChannelsRef.current.add(channel);
				}
			});
		}
	}, []);

	// 初始化和清理 — 依赖 connect/disconnect(已稳定) + enabled
	useEffect(() => {
		if (options?.enabled === false) {
			return;
		}

		// 过滤出需要WebSocket的Widget
		const wsWidgets = widgets.filter(w => isWebSocketDataSource(w.dataSource));

		if (wsWidgets.length === 0) {
			return;
		}

		connect();

		return () => {
			disconnect();
		};
		// widgets.length 作为基本类型依赖避免数组引用抖动; connect/disconnect 已稳定
	}, [widgets.length, options?.enabled, connect, disconnect]);

	return {
		connect,
		disconnect,
		refreshSubscriptions,
		connected: wsRef.current?.readyState === WebSocket.OPEN,
	};
}
