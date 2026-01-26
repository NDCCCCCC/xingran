/**
 * 动态路由组件
 * 根据后端菜单数据动态生成路由配置
 */

import { lazy, useMemo, Suspense } from 'react';
import { Routes, Route, Navigate, Outlet } from 'react-router-dom';
import { useMenuStore } from '@/store/menuStore';
import { RouteGenerator } from './routeGenerator';
import type { MenuRouteConfig } from '@/types/menu';
import Layout from '@/components/layout';
import Login from '@/pages/login';
import { Spin } from 'antd';

let cachedRoutes: MenuRouteConfig[] | null = null;
let cachedMenusCount = 0;

const hasValidToken = (): boolean => !!localStorage.getItem('accessToken');

const createLazyRoute = (route: MenuRouteConfig) => {
  const importPath = `../pages/${route.component.slice(6)}.tsx`;
  const Component = lazy(() => import(/* @vite-ignore */ importPath));

  console.log('[DynamicRoutes] Creating route:', {
    path: route.path,
    component: route.component,
    importPath,
  });

  if (route.path?.includes('port') || route.path?.includes('user') ||
      route.path?.includes('profile') || route.path?.includes('settings') ||
      route.path?.includes('notice')) {
    console.log('[DynamicRoutes] ⭐ IMPORTANT Route:', {
      path: route.path,
      component: route.component,
      importPath,
    });
  }

  if (route.path === 'dashboard') {
    return (
      <Route
        key={route.path}
        path="dashboard/*"
        element={
          <Suspense fallback={<Spin size="large" />}>
            <Component />
          </Suspense>
        }
      />
    );
  }

  return (
    <Route
      key={route.path}
      path={route.path}
      element={
        <Suspense fallback={<Spin size="large" />}>
          <Component />
        </Suspense>
      }
    />
  );
};

export function DynamicRoutes() {
  const { allMenus } = useMenuStore();

  const routeElements = useMemo(() => {
    console.log('[DynamicRoutes] allMenus:', allMenus);
    console.log('[DynamicRoutes] allMenus.length:', allMenus.length);

    if (allMenus.length === 0) {
      console.log('[DynamicRoutes] No menus, returning empty routes');
      return [];
    }

    const configs = cachedRoutes && cachedMenusCount === allMenus.length
      ? cachedRoutes
      : RouteGenerator.generate(allMenus);

    console.log('[DynamicRoutes] Generated routes:', configs);

    if (!cachedRoutes || cachedMenusCount !== allMenus.length) {
      cachedRoutes = configs;
      cachedMenusCount = allMenus.length;
    }

    return configs.map(createLazyRoute).filter(Boolean);
  }, [allMenus.length]);

  const isAuthenticated = hasValidToken();

  if (!isAuthenticated) {
    return (
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    );
  }

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
