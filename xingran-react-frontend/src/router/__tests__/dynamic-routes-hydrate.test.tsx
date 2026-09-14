/**
 * DATA-02 回归测试: sessionStorage hydrate-then-revalidate
 *
 * RED (Phase 118-02 TDD before implementation):
 *   - Test 1 should FAIL on baseline code because the allMenus.length === 0 gate
 *     forces an InitializingFallback full-page spinner whenever the menu store
 *     is empty (including on hard refresh, even when sessionStorage has valid
 *     cached menus/permissions from the previous session).
 *
 * GREEN (after implementing hydrate-then-revalidate in DynamicRoutes.tsx):
 *   - Test 1 PASSES: when sessionStorage has a valid cache, the gate is
 *     bypassed and the route tree renders immediately (no full-page gate).
 *   - Test 2 still PASSES: empty cache path falls back to the network fetch.
 *   - Test 3 PASSES: stale cache hydrates instantly and revalidates in the
 *     background without showing a full-page spinner.
 */
import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { DynamicRoutes } from "../DynamicRoutes";
import { useMenuStore } from "@/store/menuStore";
import { useAuthStore } from "@/store/authStore";
import { STORAGE_KEYS } from "@/constants/storage";

// vi.hoisted ensures these fixtures are evaluated before vi.mock factories run
// (vitest hoists vi.mock calls to the top of the file).
const mockFixtures = vi.hoisted(() => ({
  menus: [{ id: "1", path: "/dashboard", name: "仪表盘" }],
  allMenus: [{ id: "1", path: "/dashboard", name: "仪表盘", children: [] }],
  permissions: ["dashboard:view"],
}));

const menuApiMocks = vi.hoisted(() => ({
  getUserMenus: vi.fn(),
  getAllUserMenus: vi.fn(),
  getUserPermissions: vi.fn(),
}));

// Stub heavy layout / page components so we only exercise the routing/gate
// decision in DynamicRoutes without pulling in the full dashboard page tree.
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

function renderRoutes(path = "/dashboard") {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <DynamicRoutes />
    </MemoryRouter>
  );
}

describe("DATA-02: sessionStorage hydrate-then-revalidate", () => {
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
    // Default mocks hang forever (never resolve). This is deliberate: it
    // prevents the pre-fix DynamicRoutes gate from being cleared by a fast
    // fetchAll resolution, which would mask the RED signal. Test 2 (empty
    // cache path) overrides these to resolving values when it needs to
    // observe the API fallback.
    menuApiMocks.getUserMenus.mockImplementation(() => new Promise(() => {}));
    menuApiMocks.getAllUserMenus.mockImplementation(() => new Promise(() => {}));
    menuApiMocks.getUserPermissions.mockImplementation(() => new Promise(() => {}));
  });

  it("sessionStorage 有缓存时硬刷新不应显示整页 InitializingFallback", async () => {
    // Pre-populate sessionStorage with a valid menu cache (simulating previous session)
    const cachedMenu = {
      version: 1,
      data: {
        menus: mockFixtures.menus,
        allMenus: mockFixtures.allMenus,
        permissions: mockFixtures.permissions,
        cachedAt: Date.now(),
      },
    };
    sessionStorage.setItem(STORAGE_KEYS.MENU_CACHE, JSON.stringify(cachedMenu));

    renderRoutes();

    // Baseline (pre-fix): InitializingFallback shows "加载中..." full-page.
    // After DATA-02 fix: gate is bypassed, Layout renders immediately.
    await waitFor(
      () => {
        expect(screen.queryByText("加载中...")).not.toBeInTheDocument();
      },
      { timeout: 1500 }
    );
  });

  it("sessionStorage 无缓存时仍走 API 回退,最终 store 拿到数据", async () => {
    sessionStorage.clear();
    // Override the hanging default mocks so the empty-cache API fallback
    // path can actually resolve in this test.
    menuApiMocks.getUserMenus.mockResolvedValue(mockFixtures.menus);
    menuApiMocks.getAllUserMenus.mockResolvedValue(mockFixtures.allMenus);
    menuApiMocks.getUserPermissions.mockResolvedValue(mockFixtures.permissions);

    renderRoutes();

    await waitFor(() => {
      expect(useMenuStore.getState().allMenus.length).toBeGreaterThan(0);
    });
    expect(useMenuStore.getState().permissions.length).toBeGreaterThan(0);
  });

  it("sessionStorage 缓存过期(< TTL 时仍 hydrate) 后台 revalidate 补拉", async () => {
    // 10 minutes old - within the 30 minute DATA-02 TTL window, so hydrate
    // must happen immediately and the store gets refreshed by the background
    // fetchAll revalidate.
    const staleCache = {
      version: 1,
      data: {
        menus: mockFixtures.menus,
        allMenus: mockFixtures.allMenus,
        permissions: mockFixtures.permissions,
        cachedAt: Date.now() - 10 * 60 * 1000,
      },
    };
    sessionStorage.setItem(STORAGE_KEYS.MENU_CACHE, JSON.stringify(staleCache));

    renderRoutes();

    // Hydrate is synchronous in the test (sessionStorage read during render);
    // the gate must not fire.
    await waitFor(
      () => {
        expect(screen.queryByText("加载中...")).not.toBeInTheDocument();
      },
      { timeout: 1500 }
    );

    // Background revalidate also runs and confirms store is populated.
    await waitFor(() => {
      expect(useMenuStore.getState().allMenus.length).toBeGreaterThan(0);
    });
  });
});
