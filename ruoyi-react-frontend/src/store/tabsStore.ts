/**
 * 标签页状态管理
 * 用于混合式布局的多标签页功能
 */

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { TabItem } from '@/types/layout';

interface TabsState {
  tabs: TabItem[];
  activeTab: string;
  history: string[];
}

interface TabsActions {
  addTab: (tab: TabItem) => void;
  removeTab: (key: string) => void;
  setActiveTab: (key: string) => void;
  closeOtherTabs: (keepKey: string) => void;
  closeAllTabs: () => void;
  closeLeftTabs: (keepKey: string) => void;
  closeRightTabs: (keepKey: string) => void;
  updateTab: (key: string, updates: Partial<TabItem>) => void;
  pinTab: (key: string) => void;
  unpinTab: (key: string) => void;
}

type TabsStore = TabsState & TabsActions;

const MAX_TABS = 50;

export const useTabsStore = create<TabsStore>()(
  persist(
    (set, get) => ({
      // 初始状态
      tabs: [],
      activeTab: '',
      history: [],

      // 添加标签
      addTab: (tab: TabItem) => {
        const { tabs, history } = get();

        // 检查是否已存在
        const existingTab = tabs.find((t) => t.key === tab.key);
        if (existingTab) {
          // 如果已存在，激活它
          set({ activeTab: tab.key });
          return;
        }

        // 限制标签数量
        if (tabs.length >= MAX_TABS) {
          // 移除最早的非固定标签
          const closableTab = tabs.find((t) => !t.pinned && t.closable);
          if (closableTab) {
            const newTabs = tabs.filter((t) => t.key !== closableTab.key);
            const newHistory = history.filter((key) => key !== closableTab.key);
            set({
              tabs: [...newTabs, tab],
              activeTab: tab.key,
              history: [...newHistory, tab.key],
            });
          }
          return;
        }

        set({
          tabs: [...tabs, tab],
          activeTab: tab.key,
          history: [...history, tab.key],
        });
      },

      // 移除标签
      removeTab: (key: string) => {
        const { tabs, activeTab, history } = get();

        // 不允许移除固定的标签
        const tab = tabs.find((t) => t.key === key);
        if (tab?.pinned) return;

        const newTabs = tabs.filter((t) => t.key !== key);
        const newHistory = history.filter((k) => k !== key);

        // 如果移除的是当前激活的标签，切换到前一个
        let newActiveTab = activeTab;
        if (activeTab === key && newTabs.length > 0) {
          const currentIndex = history.indexOf(key);
          const previousTab = history[currentIndex - 1] || newTabs[0].key;
          newActiveTab = previousTab;
        }

        set({
          tabs: newTabs,
          activeTab: newActiveTab,
          history: newHistory,
        });
      },

      // 设置激活标签
      setActiveTab: (key: string) => {
        const { history } = get();
        set({
          activeTab: key,
          history: [...history.filter((k) => k !== key), key],
        });
      },

      // 关闭其他标签
      closeOtherTabs: (keepKey: string) => {
        const { tabs, history } = get();
        const newTabs = tabs.filter((t) => t.key === keepKey || t.pinned);
        const newHistory = history.filter((key) => key === keepKey || tabs.find((t) => t.key === key)?.pinned);

        set({
          tabs: newTabs,
          activeTab: keepKey,
          history: newHistory,
        });
      },

      // 关闭所有标签
      closeAllTabs: () => {
        const { tabs } = get();
        const pinnedTabs = tabs.filter((t) => t.pinned);

        set({
          tabs: pinnedTabs,
          activeTab: pinnedTabs.length > 0 ? pinnedTabs[0].key : '',
          history: pinnedTabs.map((t) => t.key),
        });
      },

      // 关闭左侧标签
      closeLeftTabs: (keepKey: string) => {
        const { tabs, history } = get();
        const keepIndex = tabs.findIndex((t) => t.key === keepKey);
        if (keepIndex === -1) return;

        const newTabs = tabs.filter((t, index) => {
          return index >= keepIndex || t.pinned;
        });
        const newHistory = history.filter((key) => {
          const tab = tabs.find((t) => t.key === key);
          return tab?.pinned || tabs.findIndex((t) => t.key === key) >= keepIndex;
        });

        set({
          tabs: newTabs,
          activeTab: keepKey,
          history: newHistory,
        });
      },

      // 关闭右侧标签
      closeRightTabs: (keepKey: string) => {
        const { tabs, history } = get();
        const keepIndex = tabs.findIndex((t) => t.key === keepKey);
        if (keepIndex === -1) return;

        const newTabs = tabs.filter((t, index) => {
          return index <= keepIndex || t.pinned;
        });
        const newHistory = history.filter((key) => {
          const tab = tabs.find((t) => t.key === key);
          return tab?.pinned || tabs.findIndex((t) => t.key === key) <= keepIndex;
        });

        set({
          tabs: newTabs,
          activeTab: keepKey,
          history: newHistory,
        });
      },

      // 更新标签
      updateTab: (key: string, updates: Partial<TabItem>) => {
        const { tabs } = get();
        const newTabs = tabs.map((t) =>
          t.key === key ? { ...t, ...updates } : t
        );

        set({ tabs: newTabs });
      },

      // 固定标签
      pinTab: (key: string) => {
        get().updateTab(key, { pinned: true, closable: false });
      },

      // 取消固定标签
      unpinTab: (key: string) => {
        get().updateTab(key, { pinned: false, closable: true });
      },
    }),
    {
      name: 'tabs-storage',
      partialize: (state) => ({
        tabs: state.tabs,
        activeTab: state.activeTab,
        history: state.history,
      }),
    }
  )
);

/**
 * 标签页Hook
 */
export function useTabs() {
  const {
    tabs,
    activeTab,
    history,
    addTab,
    removeTab,
    setActiveTab,
    closeOtherTabs,
    closeAllTabs,
    closeLeftTabs,
    closeRightTabs,
    updateTab,
    pinTab,
    unpinTab,
  } = useTabsStore();

  return {
    tabs,
    activeTab,
    history,
    addTab,
    removeTab,
    setActiveTab,
    closeOtherTabs,
    closeAllTabs,
    closeLeftTabs,
    closeRightTabs,
    updateTab,
    pinTab,
    unpinTab,
    hasTabs: tabs.length > 0,
  };
}
