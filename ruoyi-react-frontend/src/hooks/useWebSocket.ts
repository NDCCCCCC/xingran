/**
 * useWebSocket - WebSocket 连接管理 Hook
 *
 * 实现 WebSocket 连接管理，支持：
 * - 连接建立、断开
 * - 自动重连（指数退避策略）
 * - 消息发送和接收
 * - 连接状态管理
 */

import { useState, useRef, useCallback, useEffect } from "react";

export type WebSocketStatus = "connecting" | "connected" | "disconnected" | "error";

export interface UseWebSocketOptions {
	/** WebSocket 服务器 URL */
	url: string;
	/** 收到消息时的回调 */
	onMessage: (data: unknown) => void;
	/** 连接建立时的回调 */
	onOpen?: () => void;
	/** 连接关闭时的回调 */
	onClose?: () => void;
	/** 发生错误时的回调 */
	onError?: (error: Event) => void;
	/** 是否启用自动重连 */
	reconnect?: boolean;
	/** 初始重连间隔（毫秒） */
	reconnectInterval?: number;
	/** 最大重连次数 */
	maxReconnectAttempts?: number;
}

export interface UseWebSocketReturn {
	/** 连接状态 */
	status: WebSocketStatus;
	/** 建立连接 */
	connect: () => void;
	/** 断开连接 */
	disconnect: () => void;
	/** 发送数据 */
	send: (data: unknown) => void;
	/** 重连次数 */
	reconnectAttempts: number;
}

/**
 * WebSocket 连接管理 Hook
 */
export function useWebSocket(options?: UseWebSocketOptions): UseWebSocketReturn {
	const {
		url = "",
		onMessage = () => {},
		onOpen,
		onClose,
		onError,
		reconnect = true,
		reconnectInterval = 1000,
		maxReconnectAttempts = 10,
	} = options || {};

	const [status, setStatus] = useState<WebSocketStatus>("disconnected");
	const [reconnectAttempts, setReconnectAttempts] = useState(0);

	const wsRef = useRef<WebSocket | null>(null);
	const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
	const isManualDisconnectRef = useRef(false);

	// 清理重连定时器
	const clearReconnectTimeout = useCallback(() => {
		if (reconnectTimeoutRef.current) {
			clearTimeout(reconnectTimeoutRef.current);
			reconnectTimeoutRef.current = null;
		}
	}, []);

	// 建立连接
	const connect = useCallback(() => {
		// 如果已经连接或正在连接，不重复连接
		if (wsRef.current?.readyState === WebSocket.OPEN || wsRef.current?.readyState === WebSocket.CONNECTING) {
			return;
		}

		isManualDisconnectRef.current = false;
		setStatus("connecting");

		try {
			const ws = new WebSocket(url);

			ws.onopen = () => {
				setStatus("connected");
				setReconnectAttempts(0);
				onOpen?.();
			};

			ws.onmessage = (event) => {
				try {
					const data = JSON.parse(event.data);
					onMessage(data);
				} catch (error) {
					console.error("Failed to parse WebSocket message:", error);
				}
			};

			ws.onclose = () => {
				setStatus("disconnected");
				wsRef.current = null;
				onClose?.();

				// 自动重连（非手动断开时）
				if (reconnect && !isManualDisconnectRef.current) {
					const currentAttempts = reconnectAttempts + 1;
					if (currentAttempts < maxReconnectAttempts) {
						// 指数退避：延迟 = interval * 2^attempts，最大 30 秒
						const delay = Math.min(reconnectInterval * Math.pow(2, currentAttempts), 30000);

						reconnectTimeoutRef.current = setTimeout(() => {
							setReconnectAttempts(currentAttempts);
							connect();
						}, delay);
					} else {
						console.warn("WebSocket max reconnect attempts reached");
					}
				}
			};

			ws.onerror = (error) => {
				setStatus("error");
				console.error("WebSocket error:", error);
				onError?.(error);
			};

			wsRef.current = ws;
		} catch (error) {
			setStatus("error");
			console.error("Failed to create WebSocket connection:", error);
		}
	}, [url, onMessage, onOpen, onClose, onError, reconnect, reconnectInterval, maxReconnectAttempts, reconnectAttempts]);

	// 断开连接
	const disconnect = useCallback(() => {
		isManualDisconnectRef.current = true;
		clearReconnectTimeout();

		if (wsRef.current) {
			wsRef.current.close();
			wsRef.current = null;
		}

		setStatus("disconnected");
		setReconnectAttempts(0);
	}, [clearReconnectTimeout]);

	// 发送数据
	const send = useCallback((data: unknown) => {
		if (wsRef.current?.readyState === WebSocket.OPEN) {
			wsRef.current.send(JSON.stringify(data));
		} else {
			console.warn("WebSocket is not connected, cannot send data");
		}
	}, []);

	// 组件卸载时清理
	useEffect(() => {
		return () => {
			isManualDisconnectRef.current = true;
			clearReconnectTimeout();
			if (wsRef.current) {
				wsRef.current.close();
			}
		};
	}, [clearReconnectTimeout]);

	return {
		status,
		connect,
		disconnect,
		send,
		reconnectAttempts,
	};
}

export default useWebSocket;
