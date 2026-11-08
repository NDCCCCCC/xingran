import { create } from "zustand";
import type {
  NoticeListItem,
  WebSocketNoticeMessage,
  PortWriteProgressEvent,
  PortWriteCompletedEvent,
  ADSyncDriftAlertEvent,
} from "@/types/notice";
import type { RPAProgressMessage } from "@/types/rpa";

// RPA 进度回调类型
type RPAProgressCallback = (message: RPAProgressMessage) => void;

// P1-M4: RPA 进度监听器集合放在模块级,不进入 Zustand 响应式 state。
// 原实现把 Set 放进 state 且 onRPAProgress 直接 mutate 它,破坏不可变契约,
// 且任何订阅 state 的组件会在无关 set 时被波及。
const rpaProgressListeners = new Set<RPAProgressCallback>();

// v1.51 Phase 122 (PORTW-03): 批量写进度事件回调（progress 或 completed 二选一形状）
export type PortWriteEventCallback = (
  event: PortWriteProgressEvent | PortWriteCompletedEvent
) => void;
const portWriteListeners = new Set<PortWriteEventCallback>();

// Phase 144 (ALRT-03): AD 同步漂移告警回调。
// 同 RPA / portWrite 模式：模块级 Set，不进入 Zustand 响应式 state。
// 全局唯一 WS 连接（NotificationBell.tsx）把消息转 handleWsMessage 后派发到这里，
// 页面经 onADDriftAlert 订阅；**提示（message.warning）放在页面订阅回调里**，
// store 只负责纯派发（避免静态 message 无法消费 ConfigProvider context 的告警）。
export type ADDriftAlertCallback = (event: ADSyncDriftAlertEvent) => void;
const driftAlertListeners = new Set<ADDriftAlertCallback>();

/**
 * AD 同步漂移告警订阅（具名导出，页面直接 `import { onADDriftAlert }`）。
 * 返回退订函数；与 store action `useNoticeStore.getState().onADDriftAlert`
 * 共用同一模块级 Set（两种调用方式等价）。
 *
 * 不新建 WebSocket：消息由全局唯一 /system/ws/notices 连接转入
 * `handleWsMessage` 后派发到这里。
 */
export function onADDriftAlert(callback: ADDriftAlertCallback): () => void {
  driftAlertListeners.add(callback);
  return () => {
    driftAlertListeners.delete(callback);
  };
}

interface NoticeState {
  // 未读数量
  unreadCount: number;
  // 通知列表（用于铃铛下拉）
  notifications: NoticeListItem[];
  // 加载状态
  loading: boolean;
  // WebSocket 连接状态
  wsConnected: boolean;
}

interface NoticeActions {
  // 设置未读数量
  setUnreadCount: (count: number) => void;
  // 设置通知列表
  setNotifications: (notifications: NoticeListItem[]) => void;
  // 添加新通知到列表顶部
  addNotification: (notification: NoticeListItem) => void;
  // 标记通知为已读
  markAsRead: (noticeId: string) => void;
  // 全部标记为已读
  markAllAsRead: () => void;
  // 移除通知
  removeNotification: (noticeId: string) => void;
  // 设置加载状态
  setLoading: (loading: boolean) => void;
  // 设置 WebSocket 连接状态
  setWsConnected: (connected: boolean) => void;
  // 处理 WebSocket 消息
  handleWsMessage: (message: WebSocketNoticeMessage) => void;
  // RPA 进度事件订阅
  onRPAProgress: (callback: (message: RPAProgressMessage) => void) => () => void;
  // v1.51 Phase 122: 批量写实时进度事件订阅
  onPortWriteProgress: (callback: PortWriteEventCallback) => () => void;
  // Phase 144 ALRT-03: AD 同步漂移告警订阅（返回退订函数）
  onADDriftAlert: (callback: ADDriftAlertCallback) => () => void;
  // 重置状态
  reset: () => void;
}

const initialState: NoticeState = {
  unreadCount: 0,
  notifications: [],
  loading: false,
  wsConnected: false,
};

export const useNoticeStore = create<NoticeState & NoticeActions>()((set, get) => ({
  ...initialState,

  // 设置未读数量
  setUnreadCount: (count: number) => {
    set({ unreadCount: count });
  },

  // 设置通知列表
  setNotifications: (notifications: NoticeListItem[]) => {
    set({ notifications });
  },

  // 添加新通知到列表顶部
  addNotification: (notification: NoticeListItem) => {
    const { notifications } = get();
    // 检查是否已存在
    const exists = notifications.some((n) => n.id === notification.id);
    if (!exists) {
      set({
        notifications: [notification, ...notifications].slice(0, 50), // 最多保留50条
        unreadCount: get().unreadCount + 1,
      });
    }
  },

  // 标记通知为已读
  markAsRead: (noticeId: string) => {
    const { notifications, unreadCount } = get();
    set({
      notifications: notifications.map((n) => (n.id === noticeId ? { ...n, isRead: true } : n)),
      unreadCount: Math.max(0, unreadCount - 1),
    });
  },

  // 全部标记为已读
  markAllAsRead: () => {
    const { notifications } = get();
    set({
      notifications: notifications.map((n) => ({ ...n, isRead: true })),
      unreadCount: 0,
    });
  },

  // 移除通知
  removeNotification: (noticeId: string) => {
    const { notifications } = get();
    set({
      notifications: notifications.filter((n) => n.id !== noticeId),
    });
  },

  // 设置加载状态
  setLoading: (loading: boolean) => {
    set({ loading });
  },

  // 设置 WebSocket 连接状态
  setWsConnected: (connected: boolean) => {
    set({ wsConnected: connected });
  },

  // 处理 WebSocket 消息
  handleWsMessage: (message: WebSocketNoticeMessage) => {
    // 处理新通知
    if (message.type === "new_notice" && message.data) {
      const { noticeId, noticeTitle, noticeType, priority, createdAt } = message.data;
      get().addNotification({
        id: noticeId,
        noticeTitle,
        noticeType,
        priority,
        isRead: false,
        createdAt,
      });
    }

    // 处理 RPA 进度消息
    if (
      message.type === "rpa_progress" ||
      message.type === "rpa_completed" ||
      message.type === "rpa_failed"
    ) {
      try {
        let progressData: RPAProgressMessage;

        // 尝试从 content 字段解析
        if (message.content) {
          progressData = JSON.parse(message.content);
        } else if (
          message.data &&
          typeof message.data === "object" &&
          "executionId" in message.data
        ) {
          // 从 data 字段解析
          progressData = message.data as unknown as RPAProgressMessage;
        } else {
          return;
        }

        // 触发所有 RPA 进度监听器
        rpaProgressListeners.forEach((callback) => {
          try {
            callback(progressData);
          } catch (error) {
            console.error("[RPA Progress] Callback error:", error);
          }
        });
      } catch (error) {
        console.error("[RPA Progress] Parse error:", error);
      }
    }

    // v1.51 Phase 122 (PORTW-03): 批量写实时进度/终态分流（Content 内嵌 JSON, 仿 RPA 模式）
    if (message.type === "port_write_progress" || message.type === "port_write_completed") {
      try {
        if (!message.content) return;
        const event = JSON.parse(message.content) as
          PortWriteProgressEvent | PortWriteCompletedEvent;
        portWriteListeners.forEach((callback) => {
          try {
            callback(event);
          } catch (error) {
            console.error("[PortWrite Progress] Callback error:", error);
          }
        });
      } catch (error) {
        console.error("[PortWrite Progress] Parse error:", error);
      }
    }

    // Phase 144 (ALRT-03): AD 同步漂移告警分流。
    // 复用全局唯一 /system/ws/notices 连接，不新建第二条 WebSocket。
    // 业务 payload JSON 内嵌在 content；解析失败仅 warn 并静默跳过（不抛、不影响其它分流）。
    if (message.type === "ad_sync_drift_alert") {
      if (!message.content) {
        console.warn("[ADDriftAlert] 忽略无 content 的漂移告警 WS 消息");
        return;
      }
      let event: ADSyncDriftAlertEvent;
      try {
        event = JSON.parse(message.content) as ADSyncDriftAlertEvent;
      } catch (error) {
        console.warn("[ADDriftAlert] content 非法 JSON，忽略该消息", error);
        return;
      }
      // 兜底：即便 content 是合法 JSON 但不是对象，也不派发脏数据
      if (!event || typeof event !== "object") {
        console.warn("[ADDriftAlert] content 不是对象，忽略该消息");
        return;
      }
      // 派发给所有监听器；try/catch 隔离单个 listener 异常，避免一个订阅者崩掉全部分发
      driftAlertListeners.forEach((callback) => {
        try {
          callback(event);
        } catch (error) {
          console.error("[ADDriftAlert] Callback error:", error);
        }
      });
    }
  },

  // RPA 进度事件订阅 — 使用模块级 Set (P1-M4)
  onRPAProgress: (callback: RPAProgressCallback) => {
    rpaProgressListeners.add(callback);

    // 返回取消订阅函数
    return () => {
      rpaProgressListeners.delete(callback);
    };
  },

  // v1.51 Phase 122 (PORTW-03): 批量写实时进度事件订阅 — 同 RPA 模式（模块级 Set）
  onPortWriteProgress: (callback: PortWriteEventCallback) => {
    portWriteListeners.add(callback);
    return () => {
      portWriteListeners.delete(callback);
    };
  },

  // Phase 144 (ALRT-03): AD 同步漂移告警事件订阅 — 同 RPA / portWrite 模式（模块级 Set）。
  // 复用模块级具名导出 onADDriftAlert，保证两种调用方式共用同一 Set。
  onADDriftAlert: (callback: ADDriftAlertCallback) => onADDriftAlert(callback),

  // 重置状态。注意：模块级 listener Set 不属于 Zustand state，按 RPA / portWrite
  // 既有约定 reset() 不清空它们（订阅生命周期由各页面 useEffect cleanup 负责）。
  reset: () => {
    set(initialState);
  },
}));
