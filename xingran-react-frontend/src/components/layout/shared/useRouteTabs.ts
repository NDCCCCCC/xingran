/**
 * 路由 → 标签页跟踪 Hook（v1.23 从 HybridLayout 抽取）
 *
 * 职责：
 * - 路由变化时注册/更新 tabsStore 中的标签（含仪表盘固定标签语义）
 * - 挂载时修复仪表盘标签的 pinned/closable 状态
 * - routeConfigManager 初始化后回填各标签标题与固定状态
 *
 * ClassicLayout / HybridLayout 共用（v1.23 起 classic 也带 TabBar）。
 */

import { useEffect, useCallback } from "react";
import { useLocation } from "react-router-dom";
import { useTabs, useTabsStore } from "@/store/tabsStore";
import { useDashboardStore } from "@/store/dashboardStore";
import { routeConfigManager } from "@/router/routeConfigManager";
import { getSpecialPathTitle, matchDynamicRouteTitle, PAGE_TITLES } from "@/constants/pageTitles";

const DASHBOARD_KEY = "/dashboard";

const isDashboardPath = (path: string): boolean => {
  return path === "/dashboard" || path.startsWith("/dashboard/");
};

const getTitleByPath = (path: string): string => {
  const specialTitle = getSpecialPathTitle(path);
  if (specialTitle) {
    return specialTitle;
  }

  const dynamicTitle = matchDynamicRouteTitle(path);
  if (dynamicTitle) {
    return dynamicTitle;
  }

  if (routeConfigManager.isInitialized()) {
    return routeConfigManager.getRouteTitle(path);
  }

  const segments = path.split("/").filter(Boolean);
  return segments[segments.length - 1] || "页面";
};

export function useRouteTabs() {
  const { addTab, updateTab } = useTabs();
  const location = useLocation();
  const { currentDashboard } = useDashboardStore();

  const getDashboardTitle = useCallback((): string => {
    // 当在 /dashboard 首页时，始终显示"仪表盘"，不使用 currentDashboard.name
    if (location.pathname === "/dashboard") {
      return PAGE_TITLES.DASHBOARD;
    }
    return currentDashboard?.name || PAGE_TITLES.DASHBOARD;
  }, [currentDashboard, location.pathname]);

  useEffect(() => {
    if (location.pathname === "/login") {
      return;
    }

    const meta = routeConfigManager.isInitialized()
      ? routeConfigManager.getRouteMeta(location.pathname)
      : undefined;

    if (isDashboardPath(location.pathname)) {
      const title = getDashboardTitle();
      const state = useTabsStore.getState();
      const existingTab = state.tabs.find((t) => t.key === DASHBOARD_KEY);

      if (existingTab) {
        if (existingTab.title !== title) {
          updateTab(DASHBOARD_KEY, { title });
        }
        // 强制确保仪表盘标签是固定状态
        if (!existingTab.pinned || existingTab.closable) {
          updateTab(DASHBOARD_KEY, {
            pinned: true,
            closable: false,
          });
        }
      } else {
        // 仪表盘标签始终固定，不可关闭
        addTab({
          key: DASHBOARD_KEY,
          title: title,
          path: "/dashboard",
          closable: false,
          pinned: true,
        });
      }

      if (state.activeTab !== DASHBOARD_KEY) {
        state.setActiveTab(DASHBOARD_KEY);
      }
    } else {
      addTab({
        key: location.pathname,
        title: getTitleByPath(location.pathname),
        path: location.pathname,
        closable: !meta?.affix,
        pinned: meta?.affix,
      });
    }
  }, [location.pathname, addTab, updateTab, getDashboardTitle]);

  // 初始化时修复仪表盘标签状态
  useEffect(() => {
    const state = useTabsStore.getState();
    const dashboardTab = state.tabs.find((t) => t.key === DASHBOARD_KEY);

    // 强制修复仪表盘标签状态，确保始终固定且不可关闭
    if (dashboardTab && (dashboardTab.closable || !dashboardTab.pinned)) {
      state.updateTab(DASHBOARD_KEY, {
        pinned: true,
        closable: false,
      });
    }

    // 如果不存在仪表盘标签，创建一个
    if (!dashboardTab) {
      state.addTab({
        key: DASHBOARD_KEY,
        title: PAGE_TITLES.DASHBOARD,
        path: "/dashboard",
        closable: false,
        pinned: true,
      });
    }
  }, []);

  // 路由配置初始化后更新标签标题和固定状态
  useEffect(() => {
    if (!routeConfigManager.isInitialized()) {
      return;
    }

    const state = useTabsStore.getState();

    state.tabs.forEach((tab) => {
      // 跳过仪表盘标签，其状态已在初始化时强制设置
      if (tab.key === DASHBOARD_KEY) return;

      const newTitle = getTitleByPath(tab.path);
      const newMeta = routeConfigManager.getRouteMeta(tab.path);

      if (newTitle && newTitle !== tab.title) {
        state.updateTab(tab.key, { title: newTitle });
      }

      if (newMeta?.affix !== undefined && newMeta?.affix !== tab.pinned) {
        state.updateTab(tab.key, {
          closable: !newMeta.affix,
          pinned: newMeta.affix,
        });
      }
    });
  }, []);
}
