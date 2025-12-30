import { useEffect, useRef, useCallback } from 'react';
import { buildWebSocketUrl } from '@/lib/noticeApi';
import { useNoticeStore } from '@/store/noticeStore';
import { useAuthStore } from '@/store/authStore';
import type { WebSocketNoticeMessage } from '@/types/notice';

// 重连配置
const RECONNECT_INTERVAL = 5000; // 5秒后重连
const MAX_RECONNECT_ATTEMPTS = 10; // 最大重连次数
const HEARTBEAT_INTERVAL = 30000; // 30秒心跳

/**
 * WebSocket Hook - 用于接收实时通知推送
 */
export function useWebSocket() {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const heartbeatTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const reconnectAttemptsRef = useRef(0);

  const { wsConnected, setWsConnected, handleWsMessage, setUnreadCount } = useNoticeStore();
  const { isAuthenticated } = useAuthStore();

  // 清理定时器
  const clearTimers = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
    if (heartbeatTimerRef.current) {
      clearInterval(heartbeatTimerRef.current);
      heartbeatTimerRef.current = null;
    }
  }, []);

  // 心跳检测
  const startHeartbeat = useCallback(() => {
    clearTimers();
    heartbeatTimerRef.current = setInterval(() => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send('ping');
      }
    }, HEARTBEAT_INTERVAL);
  }, [clearTimers]);

  // 连接 WebSocket
  const connect = useCallback(() => {
    if (!isAuthenticated) {
      return;
    }

    // 如果已经连接或在连接中，不重复连接
    if (wsRef.current?.readyState === WebSocket.OPEN || wsRef.current?.readyState === WebSocket.CONNECTING) {
      return;
    }

    try {
      const url = buildWebSocketUrl();
      wsRef.current = new WebSocket(url);

      wsRef.current.onopen = () => {
        console.log('[WebSocket] 连接成功');
        setWsConnected(true);
        reconnectAttemptsRef.current = 0;
        startHeartbeat();

        // 连接成功后获取未读数量
        import('@/lib/noticeApi').then(({ getUnreadCount }) => {
          getUnreadCount()
            .then((res) => {
              setUnreadCount(res.data?.count || 0);
            })
            .catch((err) => {
              console.error('[WebSocket] 获取未读数量失败:', err);
            });
        });
      };

      wsRef.current.onmessage = (event) => {
        try {
          const data = event.data;

          // 处理心跳响应（纯文本 "pong"）
          if (data === 'pong' || data === 'ping') {
            return;
          }

          // 尝试解析为 JSON
          const message: WebSocketNoticeMessage = JSON.parse(data);

          // 处理新通知
          handleWsMessage(message);
        } catch (error) {
          console.error('[WebSocket] 解析消息失败:', error);
        }
      };

      wsRef.current.onclose = (event) => {
        console.log('[WebSocket] 连接关闭:', event.code, event.reason);
        setWsConnected(false);
        clearTimers();

        // 如果不是主动关闭，尝试重连
        if (event.code !== 1000 && isAuthenticated) {
          if (reconnectAttemptsRef.current < MAX_RECONNECT_ATTEMPTS) {
            reconnectAttemptsRef.current++;
            console.log(`[WebSocket] ${RECONNECT_INTERVAL / 1000}秒后尝试第 ${reconnectAttemptsRef.current} 次重连...`);
            reconnectTimerRef.current = setTimeout(() => {
              connect();
            }, RECONNECT_INTERVAL);
          } else {
            console.error('[WebSocket] 达到最大重连次数，停止重连');
          }
        }
      };

      wsRef.current.onerror = (error) => {
        console.error('[WebSocket] 连接错误:', error);
      };
    } catch (error) {
      console.error('[WebSocket] 创建连接失败:', error);
    }
  }, [isAuthenticated, setWsConnected, handleWsMessage, setUnreadCount, startHeartbeat, clearTimers]);

  // 断开连接
  const disconnect = useCallback(() => {
    clearTimers();
    reconnectAttemptsRef.current = 0;

    if (wsRef.current) {
      wsRef.current.close(1000, '用户主动断开');
      wsRef.current = null;
    }

    setWsConnected(false);
  }, [clearTimers, setWsConnected]);

  // 手动重连
  const reconnect = useCallback(() => {
    disconnect();
    reconnectAttemptsRef.current = 0;
    setTimeout(() => {
      connect();
    }, 100);
  }, [disconnect, connect]);

  // 组件挂载时连接，卸载时断开
  useEffect(() => {
    if (isAuthenticated) {
      connect();
    } else {
      disconnect();
    }

    return () => {
      disconnect();
    };
  }, [isAuthenticated, connect, disconnect]);

  return {
    wsConnected,
    reconnect,
    disconnect,
  };
}
