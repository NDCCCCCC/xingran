/**
 * useRealtimeUpdates - 实时更新Hook
 *
 * 通过WebSocket实现实时数据更新
 */

import { useEffect, useCallback, useRef } from "react";
import { useDashboardStore } from "@/store/dashboardStore";
import { dashboardWsUrl } from "@/lib/wsUrl";
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

/** WS 自动重连的最大次数 —— 超过后停止,避免无限重连风暴。 */
const MAX_RECONNECT_ATTEMPTS = 5;
/** 重连退避基数(毫秒)。 */
const RECONNECT_BASE_DELAY_MS = 5000;
/** 重连退避上限(毫秒)。 */
const RECONNECT_MAX_DELAY_MS = 60000;

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
export function useRealtimeUpdates(widgets: WidgetConfig[], options?: UseRealtimeUpdatesOptions) {
  const { cacheWidgetData } = useDashboardStore();
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const subscribedChannelsRef = useRef<Set<string>>(new Set());
  // 已连续重连次数 —— 达上限后停止重连
  const reconnectAttemptRef = useRef(0);

  // 获取WebSocket URL
  //
  // 路径由 API 基址派生(后端 WS 端点挂在 /api/v1/system/ws/ 下),并自动带上
  // 部署子路径前缀(生产为 /xingran)。早期硬编码的 `${host}/ws/dashboard`
  // 既缺子路径前缀、路径也与后端注册的不一致 → 握手必然 404。
  const getWsUrl = useCallback(() => dashboardWsUrl("dashboard"), []);

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
        reconnectAttemptRef.current = 0;
        options?.onConnectionChange?.(true);

        // 订阅所有WebSocket数据源的Widget
        widgets.forEach((widget) => {
          if (isWebSocketDataSource(widget.dataSource)) {
            const channel = widget.dataSource.channel;
            if (!subscribedChannelsRef.current.has(channel)) {
              ws.send(
                JSON.stringify({
                  action: "subscribe",
                  channel,
                })
              );
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
        //
        // 指数退避 + 次数上限：后端当前只实现了 /notices 端点,仪表盘频道端点
        // 尚未提供 → 握手必然失败。旧的固定 5s 重连会无限循环,在生产上表现为
        // 持续的无效连接与错误日志刷屏。
        if (options?.enabled !== false && reconnectAttemptRef.current < MAX_RECONNECT_ATTEMPTS) {
          reconnectAttemptRef.current += 1;
          const delay = Math.min(
            RECONNECT_BASE_DELAY_MS * 2 ** (reconnectAttemptRef.current - 1),
            RECONNECT_MAX_DELAY_MS
          );
          reconnectTimerRef.current = setTimeout(() => {
            connectRef.current();
          }, delay);
        } else if (reconnectAttemptRef.current >= MAX_RECONNECT_ATTEMPTS) {
          console.warn(
            "[Dashboard WebSocket] 已重连 " +
              MAX_RECONNECT_ATTEMPTS +
              " 次仍失败,停止重连" +
              "（后端尚未提供仪表盘 WS 端点时属预期,数据请走 REST 数据源）"
          );
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
    reconnectAttemptRef.current = 0;
  }, []);

  // 手动刷新订阅
  const refreshSubscriptions = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      widgets.forEach((widget) => {
        if (isWebSocketDataSource(widget.dataSource)) {
          const channel = widget.dataSource.channel;
          wsRef.current?.send(
            JSON.stringify({
              action: "subscribe",
              channel,
            })
          );
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
    const wsWidgets = widgets.filter((w) => isWebSocketDataSource(w.dataSource));

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
