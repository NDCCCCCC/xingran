/**
 * menuStore 菜单状态测试
 *
 * 覆盖:fetchMenus(缓存命中/未命中/失败)、fetchPermissions、fetchAll
 * (缓存有效/缺权限分支/forceRefresh)、setMenus/setPermissions、clearMenus、
 * invalidateCache、getCacheStatus、refreshMenuCache/clearMenuCache 导出函数。
 * TTLMenuCache 用真实实现 + resetMenuCache 复位。
 */
import { describe, it, expect, beforeEach, vi } from "vitest";

const menuApiMock = vi.hoisted(() => ({
  getUserMenus: vi.fn(),
  getAllUserMenus: vi.fn(),
  getUserPermissions: vi.fn(),
}));
vi.mock("@/lib/menuApi", () => menuApiMock);

import { useMenuStore, refreshMenuCache, clearMenuCache } from "./menuStore";
import { resetMenuCache, getMenuCache } from "@/services/cache/TTLMenuCache";

const visibleMenu = { id: "m1", menuName: "用户管理", path: "system/user" } as never;
const allMenu = { id: "m2", menuName: "隐藏菜单", path: "hidden" } as never;

function seedCache() {
  getMenuCache().setMenus([visibleMenu], [allMenu], ["system:user:list"]);
}

function resetMenuState() {
  useMenuStore.setState({
    menus: [],
    allMenus: [],
    permissions: [],
    loading: false,
    lastFetchTime: null,
    error: null,
  });
}

describe("menuStore", () => {
  let consoleSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    localStorage.clear();
    resetMenuCache();
    vi.clearAllMocks();
    menuApiMock.getUserMenus.mockResolvedValue([visibleMenu]);
    menuApiMock.getAllUserMenus.mockResolvedValue([allMenu]);
    menuApiMock.getUserPermissions.mockResolvedValue(["system:user:list"]);
    resetMenuState();
    consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    consoleSpy.mockRestore();
  });

  it("fetchMenus 无缓存时拉取并写缓存", async () => {
    await useMenuStore.getState().fetchMenus();

    expect(menuApiMock.getUserMenus).toHaveBeenCalledTimes(1);
    const state = useMenuStore.getState();
    expect(state.menus).toEqual([visibleMenu]);
    expect(state.allMenus).toEqual([allMenu]);
    expect(state.loading).toBe(false);
    expect(state.lastFetchTime).not.toBeNull();
    expect(getMenuCache().getMenus()).toEqual([visibleMenu]);
  });

  it("fetchMenus 缓存命中时不发请求", async () => {
    seedCache();
    await useMenuStore.getState().fetchMenus();

    expect(menuApiMock.getUserMenus).not.toHaveBeenCalled();
    const state = useMenuStore.getState();
    expect(state.menus).toEqual([visibleMenu]);
    expect(state.allMenus).toEqual([allMenu]);
  });

  it("fetchMenus forceRefresh 跳过缓存", async () => {
    seedCache();
    await useMenuStore.getState().fetchMenus(true);
    expect(menuApiMock.getUserMenus).toHaveBeenCalledTimes(1);
  });

  it("fetchMenus 失败:置 error 并向上抛出", async () => {
    menuApiMock.getUserMenus.mockRejectedValue(new Error("menu api down"));

    await expect(useMenuStore.getState().fetchMenus(true)).rejects.toThrow("menu api down");
    const state = useMenuStore.getState();
    expect(state.loading).toBe(false);
    expect(state.error).toBeInstanceOf(Error);
    expect(state.lastFetchTime).toBeNull();
  });

  it("fetchPermissions 无缓存时拉取并合并进缓存(保留菜单数据)", async () => {
    await useMenuStore.getState().fetchPermissions();

    expect(menuApiMock.getUserPermissions).toHaveBeenCalledTimes(1);
    expect(useMenuStore.getState().permissions).toEqual(["system:user:list"]);
    // 缓存里菜单保持为空数组(未额外拉取菜单)
    expect(getMenuCache().getAllMenus()).toEqual([]);
  });

  it("fetchPermissions 缓存命中直接使用;失败置 error", async () => {
    seedCache();
    await useMenuStore.getState().fetchPermissions();
    expect(menuApiMock.getUserPermissions).not.toHaveBeenCalled();
    expect(useMenuStore.getState().permissions).toEqual(["system:user:list"]);

    resetMenuCache();
    menuApiMock.getUserPermissions.mockRejectedValue(new Error("perm fail"));
    await expect(useMenuStore.getState().fetchPermissions(true)).rejects.toThrow("perm fail");
    expect(useMenuStore.getState().error).toBeInstanceOf(Error);
  });

  it("fetchAll 缓存完整时直接使用(不发请求)", async () => {
    seedCache();
    await useMenuStore.getState().fetchAll();

    expect(menuApiMock.getUserMenus).not.toHaveBeenCalled();
    const state = useMenuStore.getState();
    expect(state.menus).toEqual([visibleMenu]);
    expect(state.permissions).toEqual(["system:user:list"]);
  });

  it("fetchAll 缓存过期(TTL 5min)后重新拉取全量(fake timers)", async () => {
    seedCache();
    vi.useFakeTimers();
    try {
      vi.advanceTimersByTime(6 * 60 * 1000); // 超过 5 分钟 TTL
      await useMenuStore.getState().fetchAll();

      expect(menuApiMock.getUserMenus).toHaveBeenCalledTimes(1);
      expect(useMenuStore.getState().permissions).toEqual(["system:user:list"]);
    } finally {
      vi.useRealTimers();
    }
  });

  it("fetchAll 失败置 error 并抛出", async () => {
    menuApiMock.getUserPermissions.mockRejectedValue(new Error("fetch all fail"));
    await expect(useMenuStore.getState().fetchAll(true)).rejects.toThrow("fetch all fail");
    expect(useMenuStore.getState().loading).toBe(false);
  });

  it("setMenus 只更新可见菜单,保留缓存中的 allMenus/permissions", () => {
    seedCache();
    const newMenus = [{ id: "m3", menuName: "新菜单" } as never];
    useMenuStore.getState().setMenus(newMenus);

    expect(useMenuStore.getState().menus).toEqual(newMenus);
    expect(getMenuCache().getAllMenus()).toEqual([allMenu]);
    expect(getMenuCache().getPermissions()).toEqual(["system:user:list"]);
  });

  it("setPermissions 只更新权限,保留缓存中的菜单", () => {
    seedCache();
    useMenuStore.getState().setPermissions(["new:perm"]);

    expect(useMenuStore.getState().permissions).toEqual(["new:perm"]);
    expect(getMenuCache().getMenus()).toEqual([visibleMenu]);
  });

  it("clearMenus 清空状态与缓存", () => {
    seedCache();
    useMenuStore.setState({
      menus: [visibleMenu],
      allMenus: [allMenu],
      permissions: ["p"],
      lastFetchTime: 123,
      error: new Error("x"),
    });

    useMenuStore.getState().clearMenus();

    const state = useMenuStore.getState();
    expect(state.menus).toEqual([]);
    expect(state.allMenus).toEqual([]);
    expect(state.permissions).toEqual([]);
    expect(state.lastFetchTime).toBeNull();
    expect(state.error).toBeNull();
    expect(getMenuCache().isValid()).toBe(false);
  });

  it("invalidateCache 只清缓存不动状态", () => {
    seedCache();
    useMenuStore.setState({ menus: [visibleMenu] });

    useMenuStore.getState().invalidateCache();

    expect(getMenuCache().isValid()).toBe(false);
    expect(useMenuStore.getState().menus).toEqual([visibleMenu]);
  });

  it("getCacheStatus 返回缓存有效性", () => {
    useMenuStore.getState().invalidateCache();
    let status = useMenuStore.getState().getCacheStatus();
    expect(status.isValid).toBe(false);

    seedCache();
    status = useMenuStore.getState().getCacheStatus();
    expect(status.isValid).toBe(true);
    expect(status.remainingTime).toBeGreaterThan(0);
  });

  it("refreshMenuCache 导出函数 = fetchAll(true)", async () => {
    await refreshMenuCache();
    expect(menuApiMock.getUserMenus).toHaveBeenCalledTimes(1);
  });

  it("clearMenuCache 清旧 localStorage 键 + 清空状态", () => {
    localStorage.setItem("menu-storage", "{}");
    useMenuStore.setState({ menus: [visibleMenu] });

    clearMenuCache();

    expect(localStorage.getItem("menu-storage")).toBeNull();
    expect(useMenuStore.getState().menus).toEqual([]);
  });
});
