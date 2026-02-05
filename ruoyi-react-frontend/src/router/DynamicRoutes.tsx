/**
 * 动态路由组件
 * 根据后端菜单数据动态生成路由配置
 */

import { lazy, useMemo, Suspense } from 'react';
import { Routes, Route, Navigate, Outlet, useLocation } from 'react-router-dom';
import { useMenuStore } from '@/store/menuStore';
import { RouteGenerator } from './routeGenerator';
import type { MenuRouteConfig } from '@/types/menu';
import Layout from '@/components/layout';
import Login from '@/pages/login';
import { Spin } from 'antd';

// 路由缓存
let cachedRoutes: MenuRouteConfig[] | null = null;
let cachedMenusCount = 0;

// 常量定义
const DASHBOARD_PATH = 'dashboard'
const PAGES_PREFIX = 'pages/'
const PAGES_RELATIVE_PREFIX = '../pages/'
const TSX_EXTENSION = '.tsx'

// 辅助函数
const hasValidToken = (): boolean => !!localStorage.getItem('accessToken');

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

export function DynamicRoutes() {
  const { allMenus } = useMenuStore();
  const location = useLocation();

  // 检查是否是上传文件路径
  const isUploadsPath = location.pathname.startsWith('/uploads/');

  // 如果是上传文件路径，不渲染路由
  if (isUploadsPath) {
    return <UploadsFallback />;
  }

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

  const isAuthenticated = hasValidToken();

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
      <Route path="/" element={<Navigate to="/dashboard" replace />} />
      <Route element={<Layout><Outlet /></Layout>}>
        {routeElements}
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Route>
    </Routes>
  );
}

export default DynamicRoutes;
