/**
 * 标签页状态管理
 * 用于混合式布局的多标签页功能
 *
 * 持久化策略：localStorage（tab 信息非敏感，可以持久化）
 *
 * 关闭标签时的表格状态清理：removeTab / closeOtherTabs / closeAllTabs /
 * closeLeftTabs / closeRightTabs 会调用 clearTableStateByPath 清理对应路径
 * 下 sessionStorage 中的筛选/分页/排序，满足"关闭标签页才取消状态"需求。
 */

import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { TabItem } from "@/types/layout";
import { clearTableStateByPath } from "@/constants/storage";

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
  reset: () => void;
}

type TabsStore = TabsState & TabsActions;

const MAX_TABS = 50;
const DASHBOARD_KEY = "/dashboard";

// 检查是否为仪表盘标签
const isDashboardTab = (key: string): boolean => key === DASHBOARD_KEY;

export const useTabsStore = create<TabsStore>()(
  persist(
    (set, get) => ({
      // 初始状态
      tabs: [],
      activeTab: "",
      history: [],

      // 添加标签
      addTab: (tab: TabItem) => {
        const { tabs, history } = get();

        // 仪表盘标签始终固定，不可关闭
        if (isDashboardTab(tab.key)) {
          tab.pinned = true;
          tab.closable = false;
        }

        // 检查是否已存在
        const existingTab = tabs.find((t) => t.key === tab.key);
        if (existingTab) {
          // 如果已存在，激活它，并确保状态正确
          set({ activeTab: tab.key });
          // 对于仪表盘标签，确保始终是固定状态
          if (isDashboardTab(tab.key) && (!existingTab.pinned || existingTab.closable)) {
            const updatedTabs = tabs.map((t) =>
              t.key === tab.key ? { ...t, pinned: true, closable: false } : t
            );
            // 重新排序：固定标签在前
            const sortedTabs = updatedTabs.sort((a, b) => {
              if (a.pinned && !b.pinned) return -1;
              if (!a.pinned && b.pinned) return 1;
              return 0;
            });
            set({ tabs: sortedTabs });
          }
          return;
        }

        // 限制标签数量
        if (tabs.length >= MAX_TABS) {
          // 移除最早的非固定标签
          const closableTab = tabs.find((t) => !t.pinned && t.closable);
          if (closableTab) {
            // 标签被淘汰，清理其对应路径的表格状态
            clearTableStateByPath(closableTab.key);
            const newTabs = tabs.filter((t) => t.key !== closableTab.key);
            const newHistory = history.filter((key) => key !== closableTab.key);
            // 排序：固定标签在前，非固定标签在后
            const sortedTabs = [...newTabs, tab].sort((a, b) => {
              if (a.pinned && !b.pinned) return -1;
              if (!a.pinned && b.pinned) return 1;
              return 0;
            });
            set({
              tabs: sortedTabs,
              activeTab: tab.key,
              history: [...newHistory, tab.key],
            });
          }
          return;
        }

        // 排序：固定标签在前，非固定标签在后
        const sortedTabs = [...tabs, tab].sort((a, b) => {
          if (a.pinned && !b.pinned) return -1;
          if (!a.pinned && b.pinned) return 1;
          return 0;
        });

        set({
          tabs: sortedTabs,
          activeTab: tab.key,
          history: [...history, tab.key],
        });
      },

      // 移除标签
      removeTab: (key: string) => {
        // 不允许移除仪表盘标签
        if (isDashboardTab(key)) return;

        const { tabs, activeTab, history } = get();

        // 不允许移除固定的标签
        const tab = tabs.find((t) => t.key === key);
        if (tab?.pinned) return;

        // 清理该标签对应路径的表格状态（筛选/分页/排序）
        clearTableStateByPath(key);

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
        // 清理被关闭标签对应路径的表格状态（保留 keepKey 与固定标签）
        tabs
          .filter((t) => t.key !== keepKey && !t.pinned)
          .forEach((t) => clearTableStateByPath(t.key));
        const newTabs = tabs.filter((t) => t.key === keepKey || t.pinned);
        const newHistory = history.filter(
          (key) => key === keepKey || tabs.find((t) => t.key === key)?.pinned
        );

        set({
          tabs: newTabs,
          activeTab: keepKey,
          history: newHistory,
        });
      },

      // 关闭所有标签
      closeAllTabs: () => {
        const { tabs } = get();
        // 清理所有非固定标签对应路径的表格状态
        tabs.filter((t) => !t.pinned).forEach((t) => clearTableStateByPath(t.key));
        const pinnedTabs = tabs.filter((t) => t.pinned);

        set({
          tabs: pinnedTabs,
          activeTab: pinnedTabs.length > 0 ? pinnedTabs[0].key : "",
          history: pinnedTabs.map((t) => t.key),
        });
      },

      // 关闭左侧标签
      closeLeftTabs: (keepKey: string) => {
        const { tabs, history } = get();
        const keepIndex = tabs.findIndex((t) => t.key === keepKey);
        if (keepIndex === -1) return;

        // 清理被关闭标签对应路径的表格状态
        tabs
          .filter((t, index) => index < keepIndex && !t.pinned)
          .forEach((t) => clearTableStateByPath(t.key));
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

        // 清理被关闭标签对应路径的表格状态
        tabs
          .filter((t, index) => index > keepIndex && !t.pinned)
          .forEach((t) => clearTableStateByPath(t.key));
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

        // 对于仪表盘标签，强制设置为不可关闭且固定
        const finalUpdates = isDashboardTab(key)
          ? { ...updates, pinned: true, closable: false }
          : updates;

        const newTabs = tabs.map((t) => (t.key === key ? { ...t, ...finalUpdates } : t));

        // 如果更新了 pinned 状态，需要重新排序
        const updatedTab = newTabs.find((t) => t.key === key);
        if (updatedTab && "pinned" in updates) {
          // 重新排序：固定标签在前，非固定标签在后
          const sortedTabs = newTabs.sort((a, b) => {
            if (a.pinned && !b.pinned) return -1;
            if (!a.pinned && b.pinned) return 1;
            return 0;
          });
          set({ tabs: sortedTabs });
        } else {
          set({ tabs: newTabs });
        }
      },

      // 固定标签
      pinTab: (key: string) => {
        // 仪表盘标签始终固定，不需要手动固定
        if (isDashboardTab(key)) return;
        get().updateTab(key, { pinned: true, closable: false });
      },

      // 取消固定标签
      unpinTab: (key: string) => {
        // 不允许取消仪表盘标签的固定状态
        if (isDashboardTab(key)) return;
        get().updateTab(key, { pinned: false, closable: true });
      },

      // 重置状态
      reset: () => {
        set({
          tabs: [],
          activeTab: "",
          history: [],
        });
      },
    }),
    {
      name: "tabs-storage",
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
    reset,
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
    reset,
    hasTabs: tabs.length > 0,
  };
}
