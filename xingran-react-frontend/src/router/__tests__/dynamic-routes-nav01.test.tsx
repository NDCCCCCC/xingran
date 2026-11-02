/**
 * NAV-01 回归测试：F5 刷新不应回退 /dashboard（v1.24 Phase 81）
 *
 * 背景（双研究定位的根因链，见 .planning/research/ARCHITECTURE.md §4）：
 *   DynamicRoutes 首帧菜单尚未水合（allMenus=[]）→ routeElements=[] → 深层 URL
 *   无路由可匹配 → 命中 `path="*"` catch-all → <Navigate to="/dashboard" replace/>
 *   → saveLastPath 把 /dashboard 反写回 LAST_PATH（自毁循环）→ 下次刷新又从
 *   dashboard 开始。根路径 lastPath 还原机制因此失效。
 *
 * 本测试锁定两个行为：
 *   1. 菜单未就绪 + 深层 URL → **URL 保持原样**，LAST_PATH **不被覆写为 /dashboard**
 *   2. 菜单已就绪 + 无效路径 → **仍跳 /dashboard**（防过度修复导致无效路径卡死）
 *
 * 与既有 dynamic-routes-hydrate.test.tsx 的区别：那份锁「有缓存时硬刷新不显示
 * 整页 InitializingFallback」（DATA-02 hydrate-then-revalidate），本份锁 catch-all
 * 兜底的**语义区分**（未就绪 ≠ 无权限）。
 */
import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router-dom";
import { DynamicRoutes } from "../DynamicRoutes";
import { useMenuStore } from "@/store/menuStore";
import { useAuthStore } from "@/store/authStore";
import { STORAGE_KEYS } from "@/constants/storage";

const menuApiMocks = vi.hoisted(() => ({
  getUserMenus: vi.fn(),
  getAllUserMenus: vi.fn(),
  getUserPermissions: vi.fn(),
}));

vi.mock("@/components/layout", () => ({
  default: ({ children }: { children?: React.ReactNode }) => (
    <div data-testid="layout-shell">{children}</div>
  ),
}));
vi.mock("@/pages/login", () => ({
  default: () => <div data-testid="login-page">登录</div>,
}));
vi.mock("@/pages/system/notice/detail", () => ({
  default: () => <div data-testid="admin-notice-detail" />,
}));
vi.mock("@/pages/my-notices/detail", () => ({
  default: () => <div data-testid="my-notice-detail" />,
}));
vi.mock("../componentLoader", () => ({
  createLazyComponent: () => {
    const Stub = () => <div data-testid="lazy-page-stub" />;
    return Stub;
  },
}));
vi.mock("@/lib/menuApi", () => ({
  getUserMenus: menuApiMocks.getUserMenus,
  getAllUserMenus: menuApiMocks.getAllUserMenus,
  getUserPermissions: menuApiMocks.getUserPermissions,
}));

/** 暴露当前 location，用于断言 catch-all 兜底是否改写了 URL */
function LocationProbe() {
  const location = useLocation();
  return <div data-testid="location-probe">{location.pathname}</div>;
}

function renderRoutes(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <DynamicRoutes />
      <LocationProbe />
    </MemoryRouter>
  );
}

const DASHBOARD_MENU = {
  id: "1",
  path: "/dashboard",
  menuName: "仪表盘",
  menuType: "C",
  component: "pages/dashboard/index",
  children: [],
};

/**
 * 目标深层页菜单 —— F5 刷新的真实场景要求 **sessionStorage 缓存里含目标菜单**：
 * 刷新后 menuStore 无 persist（store 必空）→ hydrate 后 allMenus 才有内容；
 * 若缓存里没有目标菜单，hydrate 完成后该路径确实无路由 → 跳 /dashboard 是**正确行为**，
 * 测试就分不出修复前后。
 *
 * ⚠️ 字段名必须是 `menuName`（不是 `name`）——RouteGenerator.validateMenu 要求
 * `id` + `menuName`，缺任一则该菜单被静默跳过 → 无路由 → catch-all 命中。
 */
const USER_MENU = {
  id: "2",
  path: "/system/user",
  menuName: "系统用户",
  menuType: "C",
  component: "pages/system/user/index",
  // meta 存在但无 permissions → routeConfigManager.hasPermission 放行（向后兼容口径）
  meta: {},
  children: [],
};

/**
 * 预置 sessionStorage 菜单缓存 —— **这是 F5 刷新的真实场景**：
 * 上次会话写过菜单缓存，但 menuStore 无 persist（刷新后 store 必为空），
 * hydrate 在 useEffect 里执行 → 首帧 allMenus=[] 且 routeElements=[]。
 *
 * ⚠️ 若不预置缓存，组件会在 DynamicRoutes.tsx:246 提前 return InitializingFallback，
 * 根本不渲染 <Routes> —— catch-all 压根不挂载，测的就不是 NAV-01 的语义区分了。
 */
function seedMenuCache() {
  sessionStorage.setItem(
    STORAGE_KEYS.MENU_CACHE,
    JSON.stringify({
      version: 1,
      data: {
        menus: [DASHBOARD_MENU, USER_MENU],
        allMenus: [DASHBOARD_MENU, USER_MENU],
        permissions: ["dashboard:view", "system:user:list"],
        cachedAt: Date.now(),
      },
    })
  );
}

describe("NAV-01: F5 刷新不应回退 /dashboard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useAuthStore.setState({
      user: null,
      isAuthenticated: true,
      initialized: true,
    });
    useMenuStore.setState({
      menus: [],
      allMenus: [],
      permissions: [],
      loading: false,
      lastFetchTime: null,
      error: null,
    });
    sessionStorage.clear();
    // 默认挂起：模拟「首帧菜单尚未水合」的窗口（NAV-01 的关键时序）
    menuApiMocks.getUserMenus.mockImplementation(() => new Promise(() => {}));
    menuApiMocks.getAllUserMenus.mockImplementation(() => new Promise(() => {}));
    menuApiMocks.getUserPermissions.mockImplementation(() => new Promise(() => {}));
  });

  it("菜单未就绪 + 深层 URL：URL 不被 catch-all 改写为 /dashboard", async () => {
    // 关键：预置缓存 → 走到 <Routes> 分支，catch-all 才会挂载
    seedMenuCache();

    renderRoutes("/system/user");

    const probe = screen.getByTestId("location-probe");
    // 首帧与后续渲染都必须停在原路径（catch-all 不得抢先跳 dashboard）
    expect(probe.textContent).toBe("/system/user");
    await waitFor(() => {
      expect(screen.getByTestId("location-probe").textContent).toBe("/system/user");
    });
    expect(screen.getByTestId("location-probe").textContent).not.toBe("/dashboard");
  });

  it("菜单未就绪：LAST_PATH 不被覆写成 /dashboard（防自毁反写）", async () => {
    seedMenuCache();
    // 预置用户真实落点（模拟上一次会话停在 /system/user）
    sessionStorage.setItem(STORAGE_KEYS.LAST_PATH, "/system/user");

    renderRoutes("/system/user");

    await waitFor(() => {
      expect(screen.getByTestId("location-probe").textContent).toBe("/system/user");
    });

    // 关键：不得被兜底跳转产生的 /dashboard 覆写
    expect(sessionStorage.getItem(STORAGE_KEYS.LAST_PATH)).toBe("/system/user");
    expect(sessionStorage.getItem(STORAGE_KEYS.LAST_PATH)).not.toBe("/dashboard");
  });

  it("菜单已就绪 + 无效路径：仍跳 /dashboard（不回归）", async () => {
    // 菜单已就绪（store 有数据），且 API 挂起保证不会重置 store
    useMenuStore.setState({
      menus: [DASHBOARD_MENU],
      allMenus: [DASHBOARD_MENU],
      permissions: ["dashboard:view"],
      loading: false,
      lastFetchTime: null,
      error: null,
    });
    seedMenuCache();

    renderRoutes("/no-such-page");

    // 无效路径在菜单就绪时属于「真的无此路由」→ 兜底跳 dashboard
    await waitFor(() => {
      expect(screen.getByTestId("location-probe").textContent).toBe("/dashboard");
    });
  });
});
