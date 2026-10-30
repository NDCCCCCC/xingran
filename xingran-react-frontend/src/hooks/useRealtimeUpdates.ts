/**
 * useRealtimeUpdates - 实时更新Hook
 *
 * 通过WebSocket实现实时数据更新
 */

import { useEffect, useCallback, useRef } from "react";
import { useDashboardStore } from "@/store/dashboardStore";
import type { WidgetConfig, WebSocketDataSourceConfig, DataSourceConfig } from "@/types/dashboard";

// 类型守卫：检查 dataSource 是否有直接的 type 属性
function _hasDataSourceType(ds: DataSourceConfig): ds is WebSocketDataSourceConfig {
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

	// 获取WebSocket URL
	const getWsUrl = useCallback(() => {
		const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
		const host = import.meta.env.VITE_WS_HOST || window.location.host;
		const basePath = import.meta.env.VITE_WS_BASE_PATH || "/ws";
		return `${protocol}//${host}${basePath}/dashboard`;
	}, []);

	// connect 的 latest-ref 桥接：ws.onclose 里的自动重连需要引用 connect 自身，
	// 直接自引用会形成「声明前访问」（react-hooks/immutability）；ref 写入收敛在 effect 内
	const connectRef = useRef<() => void>(() => {});

	// 连接WebSocket
	const connect = useCallback(() => {
		if (wsRef.current?.readyState === WebSocket.OPEN) {
			return;
		}

		try {
			const ws = new WebSocket(getWsUrl());

			ws.onopen = () => {
				options?.onConnectionChange?.(true);

				// 订阅所有WebSocket数据源的Widget
				widgets.forEach((widget) => {
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
						options?.onMessage?.(widgetId, data);
					}
				} catch (error) {
					console.error("[Dashboard WebSocket] Message parse error:", error);
				}
			};

			ws.onerror = (error) => {
				console.error("[Dashboard WebSocket] Error:", error);
				options?.onError?.(error);
				options?.onConnectionChange?.(false);
			};

			ws.onclose = () => {
				options?.onConnectionChange?.(false);
				subscribedChannelsRef.current.clear();

				// 自动重连（经 ref 间接调用 connect，避免自引用）
				if (options?.enabled !== false) {
					reconnectTimerRef.current = setTimeout(() => {
						connectRef.current();
					}, 5000);
				}
			};

			wsRef.current = ws;
		} catch (error) {
			console.error("[Dashboard WebSocket] Connection error:", error);
		}
	}, [widgets, options, getWsUrl, cacheWidgetData]);

	// 同步 connect 最新引用到 ref（effect 内写入，compiler 合规）
	useEffect(() => {
		connectRef.current = connect;
	}, [connect]);

	// 断开连接
	const disconnect = useCallback(() => {
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

	// 手动刷新订阅
	const refreshSubscriptions = useCallback(() => {
		if (wsRef.current?.readyState === WebSocket.OPEN) {
			widgets.forEach((widget) => {
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
	}, [widgets]);

	// 初始化和清理
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
	}, [widgets, options?.enabled, connect, disconnect]);

	return {
		connect,
		disconnect,
		refreshSubscriptions,
		// 注意：不返回渲染期读取 wsRef 的 connected 字段（react-hooks/refs 禁止渲染期读 ref；
		// 该 hook 当前无消费方，如需连接状态请改用 state 跟踪）
	};
}
