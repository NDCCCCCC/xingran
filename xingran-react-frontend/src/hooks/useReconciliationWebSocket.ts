/**
 * useReconciliationWebSocket — 资产对账 critical 事件 WS 订阅 Hook (Phase 43 R2 / D-A4-01~04)
 *
 * 业务说明:
 *   - 复用项目现有 WS endpoint `/system/ws/notices`(同 noticeApi.buildWebSocketUrl)
 *   - 只过滤 critical_exception_detected / critical_workorder_created 2 类事件(D-A4-02)
 *   - 收到事件后自动 queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.all })
 *     触发 dashboard / 异常列表所有 useQuery 重新拉取
 *   - 暴露 onCriticalEvent 给上层做 toast 提示 / Badge 闪烁等 UI 反馈
 *   - 自动断线重连复用 useWebSocket 内置指数退避(10 次 / 30s 上限)
 *
 * 使用示例(dashboard):
 *   const queryClient = useQueryClient();
 *   const { status } = useReconciliationWebSocket({
 *     queryClient,
 *     onCriticalEvent: (event, data) => message.info(`收到 ${event}: ${data.title || data.conflict_type}`),
 *   });
 *
 * 设计考量:
 *   - queryClient 参数必须显式传入(避免循环依赖 hooks/lib)
 *   - 事件过滤在 hook 内部完成,组件只看到过滤后的 critical 事件
 *   - 非 critical 事件(new_notice / rpa_* 等)直接忽略,不影响业务
 */

import { useCallback, useEffect, useMemo } from "react";
import type { QueryClient } from "@tanstack/react-query";
import { useWebSocket, type WebSocketStatus } from "./useWebSocket";
import { buildWebSocketUrl } from "@/lib/noticeApi";
import { queryKeys } from "@/lib/queryKeys";

/** R2 critical 事件类型常量(与后端 reconciliation_workorder.go 对齐) */
export type ReconciliationCriticalEvent =
	| "critical_exception_detected"
	| "critical_workorder_created";

/** critical 事件 payload(data 字段) */
export interface ReconciliationCriticalPayload {
	workorder_id?: string;
	exception_id: string;
	asset_id?: string;
	asset_code?: string;
	conflict_type?: string;
	severity: string;
	title?: string;
	detected_at?: string;
}

/** hook 选项 */
export interface UseReconciliationWebSocketOptions {
	/** TanStack Query queryClient(必填,显式传入避免循环依赖) */
	queryClient: QueryClient;
	/** critical 事件回调(用于 toast / Badge 闪烁等) */
	onCriticalEvent?: (
		event: ReconciliationCriticalEvent,
		data: ReconciliationCriticalPayload
	) => void;
	/** 是否启用订阅(默认 true) */
	enabled?: boolean;
}

/** hook 返回值 */
export interface UseReconciliationWebSocketReturn {
	/** WS 连接状态(connecting / connected / disconnected / error) */
	status: WebSocketStatus;
	/** 手动断开 */
	disconnect: () => void;
}

/**
 * 订阅 critical 异常/工单 WS 事件 + 自动 query invalidation
 */
export function useReconciliationWebSocket(
	options: UseReconciliationWebSocketOptions
): UseReconciliationWebSocketReturn {
	const { queryClient, onCriticalEvent, enabled = true } = options;

	// 缓存 callback 引用(useMemo 避免子组件 useEffect 重新触发)
	const onCriticalEventRef = useMemo(() => onCriticalEvent, [onCriticalEvent]);

	// 事件处理:解析 content JSON,过滤 critical_* 事件,触发 query invalidation + 回调
	const handleMessage = useCallback(
		(raw: unknown) => {
			// 1. 容错:必须能 JSON.parse(NoticeHub 已 JSON 编码消息)
			const msg = raw as { type?: string; content?: string } | null;
			if (!msg || typeof msg !== "object") return;
			const eventType = msg.type;
			if (!eventType) return;

			// 2. 过滤 critical 2 类事件(D-A4-02)
			if (
				eventType !== "critical_exception_detected" &&
				eventType !== "critical_workorder_created"
			) {
				return;
			}

			// 3. 解析 content(JSON 字符串 → payload)
			let payload: ReconciliationCriticalPayload | undefined;
			if (typeof msg.content === "string" && msg.content.length > 0) {
				try {
					payload = JSON.parse(msg.content) as ReconciliationCriticalPayload;
				} catch {
					// content 不是 JSON,降级为空 payload(只回调 event type)
					payload = undefined;
				}
			}

			// 4. 触发 query invalidation(refetch dashboard + 异常列表)
			// 使用 queryKeys.reconciliation.all 前缀,匹配所有 reconciliation 子键
			queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.all });

			// 5. 业务回调(toast / Badge)
			if (payload) {
				onCriticalEventRef?.(
					eventType as ReconciliationCriticalEvent,
					payload
				);
			}
		},
		[queryClient, onCriticalEventRef]
	);

	// 计算 WS URL(enabled=false 时传空字符串,useWebSocket 不会尝试连接)
	const wsUrl = useMemo(() => (enabled ? buildWebSocketUrl() : ""), [enabled]);

	const { status, disconnect } = useWebSocket({
		url: wsUrl,
		onMessage: handleMessage,
		reconnect: enabled,
		reconnectInterval: 2000,
		maxReconnectAttempts: 10,
	});

	// 组件卸载时主动断开(useWebSocket 自身也会清理,这里显式调用增强语义)
	useEffect(() => {
		return () => {
			disconnect();
		};
	}, [disconnect]);

	return {
		status,
		disconnect,
	};
}

export default useReconciliationWebSocket;