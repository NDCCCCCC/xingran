import { useEffect } from 'react';
import { buildWebSocketUrl } from '@/lib/noticeApi';
import { useNoticeStore } from '@/store/noticeStore';
import { useAuthStore } from '@/store/authStore';
import type { WebSocketNoticeMessage } from '@/types/notice';

// 重连配置
const RECONNECT_INTERVAL = 5000; // 5秒后重连
const MAX_RECONNECT_ATTEMPTS = 10; // 最大重连次数
const HEARTBEAT_INTERVAL = 30000; // 30秒心跳

// ========================================
// 模块级单例 - 全局只有一个 WebSocket 连接
// ========================================

let wsInstance: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let heartbeatTimer: ReturnType<typeof setInterval> | null = null;
let reconnectAttempts = 0;
let authStateUnsubscribe: (() => void) | null = null;

// 清理定时器
function clearTimers() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  if (heartbeatTimer) {
    clearInterval(heartbeatTimer);
    heartbeatTimer = null;
  }
}

// 心跳检测
function startHeartbeat() {
  clearTimers();
  heartbeatTimer = setInterval(() => {
    if (wsInstance?.readyState === WebSocket.OPEN) {
      wsInstance.send('ping');
    }
  }, HEARTBEAT_INTERVAL);
}

// 连接 WebSocket
function connect() {
  const { isAuthenticated } = useAuthStore.getState();
  const { setWsConnected, handleWsMessage, setUnreadCount } = useNoticeStore.getState();

  if (!isAuthenticated) {
    return;
  }

  // 如果已经连接或在连接中，不重复连接
  if (wsInstance?.readyState === WebSocket.OPEN || wsInstance?.readyState === WebSocket.CONNECTING) {
    return;
  }

  try {
    const url = buildWebSocketUrl();
    wsInstance = new WebSocket(url);

    wsInstance.onopen = () => {
      setWsConnected(true);
      reconnectAttempts = 0;
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

    wsInstance.onmessage = (event) => {
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

    wsInstance.onclose = (event) => {
      setWsConnected(false);
      clearTimers();

      // 如果不是主动关闭，尝试重连
      if (event.code !== 1000 && isAuthenticated) {
        if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
          reconnectAttempts++;
          reconnectTimer = setTimeout(() => {
            connect();
          }, RECONNECT_INTERVAL);
        }
      }
    };

    wsInstance.onerror = () => {
      // WebSocket 连接错误，静默处理
    };
  } catch (error) {
    // WebSocket 创建失败，静默处理
  }
}

// 断开连接
function disconnect() {
  clearTimers();
  reconnectAttempts = 0;

  if (wsInstance) {
    wsInstance.close(1000, '用户主动断开');
    wsInstance = null;
  }

  useNoticeStore.getState().setWsConnected(false);
}

// 手动重连
function reconnect() {
  disconnect();
  reconnectAttempts = 0;
  setTimeout(() => {
    connect();
  }, 100);
}

/**
 * WebSocket Hook - 用于接收实时通知推送
 *
 * 使用模块级单例模式，确保全局只有一个 WebSocket 连接
 */
export function useWebSocket() {
  const wsConnected = useNoticeStore((state) => state.wsConnected);

  // 初始化监听认证状态变化（组件挂载时执行一次）
  useEffect(() => {
    // 如果已经订阅过，跳过
    if (authStateUnsubscribe) {
      return;
    }

    // 监听认证状态变化
    // zustand 的 subscribe 会立即调用一次回调函数并传入当前值
    authStateUnsubscribe = useAuthStore.subscribe(
      (state) => {
        if (state.isAuthenticated) {
          connect();
        } else {
          disconnect();
        }
      }
    );

    return () => {
      // 组件卸载时清理（但不关闭 WebSocket 连接，因为其他组件可能还在使用）
    };
  }, []); // 空依赖数组，只运行一次

  return {
    wsConnected,
    reconnect,
    disconnect,
  };
}
