/**
 * 动态路由组件
 * 根据后端菜单数据动态生成路由配置
 */

import { useMemo, Suspense, useEffect, useState } from "react";
import { Routes, Route, Navigate, Outlet, useLocation } from "react-router-dom";
import { useMenuStore } from "@/store/menuStore";
import { useAuthStore } from "@/store/authStore";
import { RouteGenerator } from "./routeGenerator";
import { routeConfigManager } from "./routeConfigManager";
import { createLazyComponent } from "./componentLoader";
import type { MenuRouteConfig } from "@/types/menu";
import Layout from "@/components/layout";
import Login from "@/pages/login";
import AdminNoticeDetailPage from "@/pages/system/notice/detail";
import MyNoticeDetailPage from "@/pages/my-notices/detail";
import { Spin } from "antd";
import { STORAGE_KEYS } from "@/constants/storage";

// Get last visited path from sessionStorage
export const getLastPath = (): string | null => {
  try {
    return sessionStorage.getItem(STORAGE_KEYS.LAST_PATH);
  } catch {
    return null;
  }
};

// Save current path to sessionStorage
export const saveLastPath = (path: string): void => {
  try {
    // 保存所有路径，除了 /login
    if (path && path !== "/login") {
      sessionStorage.setItem(STORAGE_KEYS.LAST_PATH, path);
    }
  } catch {
    // Ignore sessionStorage errors
  }
};

// Clear last path (for logout)
export const clearLastPath = (): void => {
  try {
    sessionStorage.removeItem(STORAGE_KEYS.LAST_PATH);
  } catch {
    // Ignore sessionStorage errors
  }
};

// 路由缓存（移入组件 useRef,避免模块级可变状态被组件改写）
// 注意:这意味着每个 DynamicRoutes 实例有独立缓存;如需跨实例共享,需要提升到 store

const DASHBOARD_PATH = "dashboard";

const createLazyRoute = (route: MenuRouteConfig) => {
  const Component = createLazyComponent(route.component);

  const fallback = <Spin size="large" />;
  const element = (
    <Suspense fallback={fallback}>
      <Component />
    </Suspense>
  );

  if (route.path === DASHBOARD_PATH) {
    return (
      <Route
        key={route.path}
        path={`${DASHBOARD_PATH}/*`}
        element={element}
      />
    );
  }

  return (
    <Route
      key={route.path}
      path={route.path}
      element={element}
    />
  );
};

function UploadsFallback() {
  // 对于 /uploads/* 路径，返回 null 让浏览器直接处理静态资源
  // 这些请求应该由后端服务器或 Vite 代理处理
  return null;
}

function InitializingFallback() {
  return (
    <div style={{
      display: "flex",
      flexDirection: "column",
      justifyContent: "center",
      alignItems: "center",
      height: "100vh",
      fontSize: "16px",
      color: "var(--theme-text-tertiary, #999)"
    }}>
      <Spin size="large" />
      <div style={{ marginTop: 16 }}>加载中...</div>
    </div>
  );
}

export function DynamicRoutes() {
  const { allMenus, fetchAll, permissions } = useMenuStore();
  const { isAuthenticated, initialized } = useAuthStore();
  const location = useLocation();

  // 使用 state 来保存上次访问的路径
  const [lastPath, setLastPath] = useState<string | null>(() => getLastPath());

  // 兜底：防止 initialized 永远停在 false（例如 HMR 导致 onRehydrateStorage 失败）
  // 3 秒后仍未初始化，强制将状态重置为未认证，让登录页有机会渲染。
  useEffect(() => {
    if (initialized) return;
    const timer = window.setTimeout(() => {
      const current = useAuthStore.getState();
      if (current.initialized) return;
      console.warn("[DynamicRoutes] 初始化超时，强制重置为未认证");
      useAuthStore.setState({
        user: null,
        isAuthenticated: false,
        menusLoaded: false,
        initialized: true,
      });
    }, 3000);
    return () => window.clearTimeout(timer);
  }, [initialized]);

  // 当用户已认证但菜单未加载时，自动加载菜单
  useEffect(() => {
    if (isAuthenticated && initialized && allMenus.length === 0) {
      fetchAll().catch((error) => {
        console.error("Failed to load menus after refresh:", error);
      });
    }
  }, [isAuthenticated, initialized, allMenus.length, fetchAll]);

  // P0-1: 菜单变化时初始化 routeConfigManager(routeMap)
  // 为其他消费者(useTabSync 等)提供 routeTitle/breadcrumb 服务。
  // 本组件的权限检查走 inline 逻辑(见下方的 useMemo), 不依赖该单例。
  useEffect(() => {
    if (allMenus.length > 0) {
      routeConfigManager.initialize(allMenus);
    }
  }, [allMenus]);

  const routeElements = useMemo(() => {
    if (allMenus.length === 0) {
      return [];
    }

    const configs = RouteGenerator.generate(allMenus);

    // P0-1: 前端路由权限第二层防线 (inline 避免调用模块级单例)
    // 后端已按角色 RBAC 过滤 allMenus(第一层), 此处对 menu.meta.permissions
    // 做细粒度二次校验。meta.permissions 为空的路由直接放行(向后兼容)。
    return configs
      .map((route) => {
        const requiredPerms = route.meta?.permissions;
        const allowed =
          !requiredPerms || requiredPerms.length === 0 ||
          requiredPerms.some((p: string) => permissions.includes(p));
        if (allowed) {
          return createLazyRoute(route);
        }
        // 无权限的路由重定向到 dashboard (避免空白页)
        const missing = requiredPerms.filter((p: string) => !permissions.includes(p));
        console.warn(`[RouteGuard] ${route.path} 无权限,缺少: ${missing.join(", ")}`);
        return (
          <Route
            key={route.path}
            path={route.path}
            element={<Navigate to="/dashboard" replace />}
          />
        );
      })
      .filter(Boolean);
    // permissions 是从 store 读取的, 变化时 zustand 会触发 re-render,
    // 此处显式声明让它作为依赖, 避免 React Compiler 误报
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [allMenus, permissions]);

  // 保存当前路径到 sessionStorage
  useEffect(() => {
    if (isAuthenticated && location.pathname) {
      // 保存到 sessionStorage
      saveLastPath(location.pathname);
      // 同时更新 state
      setLastPath(location.pathname);
    }
  }, [location.pathname, isAuthenticated]);

  // 检查是否是上传文件路径
  const isUploadsPath = location.pathname.startsWith("/uploads/");

  // 如果是上传文件路径，不渲染路由
  if (isUploadsPath) {
    return <UploadsFallback />;
  }

  // 初始化中，显示 loading
  if (!initialized) {
    return <InitializingFallback />;
  }

  // 未认证用户路由
  if (!isAuthenticated) {
    return (
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    );
  }

  // 已认证用户但菜单未加载完成，显示 loading（避免跳转到 dashboard）
  if (allMenus.length === 0) {
    return <InitializingFallback />;
  }

  // 已认证用户路由
  return (
    <Routes>
      <Route path="/login" element={<Navigate to="/" replace />} />
      <Route
        path="/"
        element={
          <Navigate
            to={lastPath && lastPath !== "/" ? lastPath : "/dashboard"}
            replace
          />
        }
      />
      <Route element={<Layout><Outlet /></Layout>}>
        {routeElements}
        {/* 通知公告详情: 静态子路由,详情页无对应 sys_menu 节点,无法走 RouteGenerator */}
        <Route path="system/notice/:id" element={<AdminNoticeDetailPage />} />
        {/* 我的通知详情: 同上,NoticeBell + my-notices 列表查看按钮跳此处 */}
        <Route path="my-notices/:id" element={<MyNoticeDetailPage />} />
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Route>
    </Routes>
  );
}

export default DynamicRoutes;
