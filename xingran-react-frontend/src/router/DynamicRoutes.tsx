/**
 * 动态路由组件
 * 根据后端菜单数据动态生成路由配置
 *
 * 设计要点:
 *   - 路由权限过滤在生成阶段完成(不是渲染阶段), 避免 <Navigate> 跳转带来的
 *     "路径泄露" (恶意用户能看到 URL 已被识别但被屏蔽).
 *   - 静态详情路由(/system/notice/:id, /my-notices/:id) 通过 RouteGuard 包装,
 *     提供权限点列表 + 后端兜底校验, 避免主验证漏洞.
 *   - routeConfigManager.initialize() 移到 useMemo 之前, 消除之前 useEffect 调用
 *     "未初始化" 单例的时序问题.
 */

import { useMemo, Suspense, useEffect } from "react";
import { Routes, Route, Navigate, Outlet, useLocation } from "react-router-dom";
import { useMenuStore } from "@/store/menuStore";
import { useAuthStore } from "@/store/authStore";
import { RouteGenerator } from "./routeGenerator";
import { routeConfigManager } from "./routeConfigManager";
import { RouteGuard } from "./RouteGuard";
import { createLazyComponent } from "./componentLoader";
import type { MenuRouteConfig } from "@/types/menu";
import type { Menu } from "@/types";
import Layout from "@/components/layout";
import Login from "@/pages/login";
import AdminNoticeDetailPage from "@/pages/system/notice/detail";
import MyNoticeDetailPage from "@/pages/my-notices/detail";
import { Spin } from "antd";
import { STORAGE_KEYS, MENU_CACHE_VERSION } from "@/constants/storage";

// ==================== DATA-02: 菜单+权限 sessionStorage 缓存(hydrate-then-revalidate) ====================

/** 30 分钟 TTL,与 App.tsx react-query staleTime 同量级的后台补拉窗口 */
const MENU_CACHE_TTL_MS = 30 * 60 * 1000;

interface CachedMenuData {
  version: number;
  data: {
    menus: Menu[];
    allMenus: Menu[];
    permissions: string[];
    cachedAt: number;
  };
}

/**
 * 读取并校验 sessionStorage 菜单缓存。
 * 校验失败(version 不符 / JSON 损坏 / 过期 / 结构异常)返回 null,走 API 回退。
 */
const readMenuCache = (): CachedMenuData["data"] | null => {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEYS.MENU_CACHE);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as CachedMenuData;
    if (!parsed || parsed.version !== MENU_CACHE_VERSION) return null;
    if (!parsed.data || !Array.isArray(parsed.data.allMenus)) return null;
    if (Date.now() - parsed.data.cachedAt > MENU_CACHE_TTL_MS) return null;
    return parsed.data;
  } catch {
    return null;
  }
};

/**
 * 写入 sessionStorage 菜单缓存(revalidate 成功后调用)。
 * 静默吞掉写入异常(隐私模式 / 配额溢出),hydrate 缺失只影响下次刷新首屏速度。
 */
const writeMenuCache = (
  data: Pick<CachedMenuData["data"], "menus" | "allMenus" | "permissions">
): void => {
  try {
    const payload: CachedMenuData = {
      version: MENU_CACHE_VERSION,
      data: { ...data, cachedAt: Date.now() },
    };
    sessionStorage.setItem(STORAGE_KEYS.MENU_CACHE, JSON.stringify(payload));
  } catch {
    // 静默失败 — 下次刷新时无 hydrate 但会走 API 回退
  }
};

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
    return <Route key={route.path} path={`${DASHBOARD_PATH}/*`} element={element} />;
  }

  return <Route key={route.path} path={route.path} element={element} />;
};

function UploadsFallback() {
  // 对于 /uploads/* 路径，返回 null 让浏览器直接处理静态资源
  // 这些请求应该由后端服务器或 Vite 代理处理
  return null;
}

function InitializingFallback() {
  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        justifyContent: "center",
        alignItems: "center",
        height: "100vh",
        fontSize: "16px",
        color: "var(--theme-text-tertiary, #999)",
      }}
    >
      <Spin size="large" />
      <div style={{ marginTop: 16 }}>加载中...</div>
    </div>
  );
}

export function DynamicRoutes() {
  const allMenus = useMenuStore((s) => s.allMenus);
  const fetchAll = useMenuStore((s) => s.fetchAll);
  const permissions = useMenuStore((s) => s.permissions);
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const initialized = useAuthStore((s) => s.initialized);
  const location = useLocation();

  // 上次访问的路径直接从 sessionStorage 派生 (不再镜像到 state, 避免 effect 内同步 setState)
  const lastPath = getLastPath();

  // DATA-02: 同步读取 sessionStorage 菜单缓存,在首次渲染时就决定是否绕过整页门控
  const cachedMenuData = useMemo(() => readMenuCache(), []);

  // DATA-02: 硬刷新时立即把缓存里的菜单+权限同步到菜单 store(绕过 setMenus 触发的
  // TTLMenuCache 写入,保证后续 fetchAll 的 revalidate 仍走真实 API)。
  // 用 useMenuStore.setState 直接写状态——与本文件已有的 useAuthStore.setState
  // 兜底(初始化超时)是同一种模式。
  useEffect(() => {
    if (!cachedMenuData || !isAuthenticated) return;
    useMenuStore.setState({
      menus: cachedMenuData.menus,
      allMenus: cachedMenuData.allMenus,
      permissions: cachedMenuData.permissions,
    });
  }, [cachedMenuData, isAuthenticated]);

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

  // 已认证时加载/补拉菜单:
  // - 有 hydrate 缓存 → 后台 revalidate(外壳已渲染,完成后回写 sessionStorage)
  // - 无缓存 → 首次加载(初次渲染期间仍由下方门控显示 loading)
  useEffect(() => {
    if (!isAuthenticated || !initialized) return;
    fetchAll()
      .then(() => {
        const { menus, allMenus, permissions } = useMenuStore.getState();
        writeMenuCache({ menus, allMenus, permissions });
      })
      .catch((error) => {
        console.error("Failed to load menus after refresh:", error);
      });
  }, [isAuthenticated, initialized, fetchAll]);

  // P0-1 路由权限过滤:
  // 1. routeConfigManager.initialize 在 useMemo 内同步执行, 消除之前 useEffect
  //    调用未初始化单例的时序问题 (dev 验证捕到的 bug)
  // 2. 过滤在路由生成阶段完成, 无权限路由根本不进入 React Router tree
  //    (避免 <Navigate> 跳转时泄露路径存在性)
  // 3. meta.permissions 为空的路由默认放行 (向后兼容)
  const routeElements = useMemo(() => {
    if (allMenus.length === 0) {
      return [];
    }

    // 同步初始化 routeConfigManager (为 useTabSync 等其他消费者提供 routeTitle/breadcrumb)
    routeConfigManager.initialize(allMenus);

    const configs = RouteGenerator.generate(allMenus);

    return configs
      .filter((route) => routeConfigManager.hasPermission(route.path, permissions).hasPermission)
      .map(createLazyRoute);
  }, [allMenus, permissions]);

  // 保存当前路径到 sessionStorage (外部系统同步)
  useEffect(() => {
    if (isAuthenticated && location.pathname) {
      saveLastPath(location.pathname);
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

  // 已认证用户但菜单未加载完成且本地无 sessionStorage 缓存可水合时，显示 loading
  // DATA-02: 有缓存时不再阻塞——hydrate 后台补拉,外壳立即渲染
  //          无缓存时保持原有行为(避免跳转到 dashboard 干扰 path 还原)
  if (allMenus.length === 0 && !cachedMenuData) {
    return <InitializingFallback />;
  }

  // 已认证用户路由
  return (
    <Routes>
      <Route path="/login" element={<Navigate to="/" replace />} />
      <Route
        path="/"
        element={<Navigate to={lastPath && lastPath !== "/" ? lastPath : "/dashboard"} replace />}
      />
      <Route
        element={
          <Layout>
            <Outlet />
          </Layout>
        }
      >
        {routeElements}
        {/* 静态详情路由: 走 RouteGuard 包装而非 RouteGenerator (无法走 menu 节点路径)
         *  - AdminNoticeDetailPage 需 'system:notice:list' 权限 (列表权限隐含查看详情)
         *  - MyNoticeDetailPage 不需服务端权限 (用户自己的通知)
         *  - 后端 API 仍在校验数据归属, 客户端守卫仅 UX 优化 */}
        <Route
          path="system/notice/:id"
          element={
            <RouteGuard permissions={["system:notice:list"]} fallback="/system/notice">
              <AdminNoticeDetailPage />
            </RouteGuard>
          }
        />
        <Route
          path="my-notices/:id"
          element={
            <RouteGuard permissions={[]} fallback="/my-notices">
              <MyNoticeDetailPage />
            </RouteGuard>
          }
        />
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Route>
    </Routes>
  );
}
