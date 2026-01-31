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
import FloorPlanEditor from '@/pages/operations/floor-plan-editor';
import { Spin } from 'antd';

let cachedRoutes: MenuRouteConfig[] | null = null;
let cachedMenusCount = 0;

const hasValidToken = (): boolean => !!localStorage.getItem('accessToken');

const createLazyRoute = (route: MenuRouteConfig) => {
  // route.component 格式: pages/operations/floor-plan-editor/index
  // Vite 动态导入需要使用相对路径或绝对路径，不支持别名
  let componentPath = route.component;

  // 去掉 'pages/' 前缀（如果有）
  if (componentPath.startsWith('pages/')) {
    componentPath = componentPath.slice(6);
  }

  // 添加 .tsx 扩展名（如果没有）
  if (!componentPath.endsWith('.tsx')) {
    componentPath = `${componentPath}.tsx`;
  }

  // 使用相对路径从 router 目录到 pages 目录
  // router.tsx 位于 src/router/，pages 位于 src/pages/
  // 所以需要使用 ../pages/ 前缀
  const importPath = `../pages/${componentPath}`;

  const Component = lazy(() => import(/* @vite-ignore */ importPath));

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
        {/* 平面图编辑器路由 - 静态路由，支持楼层ID参数 */}
        <Route
          path="floor-plan-editor/:floorId"
          element={
            <Suspense fallback={<Spin size="large" />}>
              <FloorPlanEditor />
            </Suspense>
          }
        />
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Route>
    </Routes>
  );
}

export default DynamicRoutes;
