/**
 * 动态路由组件
 * 根据后端菜单数据动态生成路由配置
 */

import { lazy, useMemo, Suspense, useEffect, useState } from 'react';
import { Routes, Route, Navigate, Outlet, useLocation } from 'react-router-dom';
import { useMenuStore } from '@/store/menuStore';
import { useAuthStore } from '@/store/authStore';
import { RouteGenerator } from './routeGenerator';
import type { MenuRouteConfig } from '@/types/menu';
import Layout from '@/components/layout';
import Login from '@/pages/login';
import { Spin } from 'antd';

// sessionStorage key for storing last visited path
const LAST_PATH_KEY = 'ruoyi_last_visited_path';

// Get last visited path from sessionStorage
export const getLastPath = (): string | null => {
  try {
    return sessionStorage.getItem(LAST_PATH_KEY);
  } catch {
    return null;
  }
};

// Save current path to sessionStorage
export const saveLastPath = (path: string): void => {
  try {
    // 保存所有路径，除了 /login
    if (path && path !== '/login') {
      sessionStorage.setItem(LAST_PATH_KEY, path);
    }
  } catch {
    // Ignore sessionStorage errors
  }
};

// Clear last path (for logout)
export const clearLastPath = (): void => {
  try {
    sessionStorage.removeItem(LAST_PATH_KEY);
  } catch {
    // Ignore sessionStorage errors
  }
};

// 路由缓存
let cachedRoutes: MenuRouteConfig[] | null = null;
let cachedMenusCount = 0;

// 常量定义
const DASHBOARD_PATH = 'dashboard'
const PAGES_PREFIX = 'pages/'
const PAGES_RELATIVE_PREFIX = '../pages/'
const TSX_EXTENSION = '.tsx'

const normalizeComponentPath = (componentPath: string): string => {
  let path = componentPath

  if (path.startsWith(PAGES_PREFIX)) {
    path = path.slice(PAGES_PREFIX.length)
  }

  if (!path.endsWith(TSX_EXTENSION)) {
    path = `${path}${TSX_EXTENSION}`
  }

  return `${PAGES_RELATIVE_PREFIX}${path}`
}

const createLazyRoute = (route: MenuRouteConfig) => {
  const importPath = normalizeComponentPath(route.component);
  const Component = lazy(() => import(/* @vite-ignore */ importPath));

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
      display: 'flex',
      justifyContent: 'center',
      alignItems: 'center',
      height: '100vh',
      fontSize: '16px',
      color: '#999'
    }}>
      <Spin size="large" tip="加载中..." />
    </div>
  );
}

export function DynamicRoutes() {
  const { allMenus, fetchAll } = useMenuStore();
  const { isAuthenticated, initialized } = useAuthStore();
  const location = useLocation();

  // 使用 state 来保存上次访问的路径
  const [lastPath, setLastPath] = useState<string | null>(() => getLastPath());

  // 当用户已认证但菜单未加载时，自动加载菜单
  useEffect(() => {
    if (isAuthenticated && initialized && allMenus.length === 0) {
      fetchAll().catch((error) => {
        console.error('Failed to load menus after refresh:', error);
      });
    }
  }, [isAuthenticated, initialized, allMenus.length, fetchAll]);

  const routeElements = useMemo(() => {
    if (allMenus.length === 0) {
      return [];
    }

    const configs = cachedRoutes && cachedMenusCount === allMenus.length
      ? cachedRoutes
      : RouteGenerator.generate(allMenus);

    if (!cachedRoutes || cachedMenusCount !== allMenus.length) {
      cachedRoutes = configs;
      cachedMenusCount = allMenus.length;
    }

    return configs.map(createLazyRoute).filter(Boolean);
  }, [allMenus.length]);

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
  const isUploadsPath = location.pathname.startsWith('/uploads/');

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

  // 已认证用户路由
  return (
    <Routes>
      <Route path="/login" element={<Navigate to="/" replace />} />
      <Route path="/" element={<Navigate to={lastPath || '/dashboard'} replace />} />
      <Route element={<Layout><Outlet /></Layout>}>
        {routeElements}
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Route>
    </Routes>
  );
}

export default DynamicRoutes;
