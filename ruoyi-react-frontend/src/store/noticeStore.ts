import { create } from 'zustand';
import type { NoticeListItem, WebSocketNoticeMessage } from '@/types/notice';

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
      notifications: notifications.map((n) =>
        n.id === noticeId ? { ...n, isRead: true } : n
      ),
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
    if (message.type === 'new_notice' && message.data) {
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
  },

  // 重置状态
  reset: () => {
    set(initialState);
  },
}));
