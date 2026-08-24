/**
 * authStore 认证状态测试
 *
 * 覆盖:login 成功/失败、logout 清理链、updateUser、loadMenusAfterLogin、
 * getTokenManager、initializeFromStorage 全分支(重复初始化短路/刷新成功/
 * 刷新失败清理/无 token/fatal 兜底)。
 * TokenManager/SecureTokenStorageImpl 整模块 mock(T-83-04-01 假凭证),
 * fake timers 覆盖 TokenManager 集成时序(T-83-04-02)。
 */
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

const tokenManagerMock = vi.hoisted(() => ({
  initializeTokens: vi.fn(),
  getRefreshToken: vi.fn(),
  refreshToken: vi.fn(),
  clearTokens: vi.fn(),
}));

// 注意:new 调用需要 constructable 实现(普通 function,this 被 return 的对象覆盖)
vi.mock("@/utils/token/TokenManager", () => ({
  TokenManager: vi.fn(() => {
    return tokenManagerMock;
  }),
}));

vi.mock("@/utils/token/SecureTokenStorageImpl", () => ({
  SecureTokenStorageImpl: vi.fn(() => {
    return {};
  }),
}));

const apiPostMock = vi.hoisted(() => vi.fn());
vi.mock("@/lib/api", () => ({
  post: apiPostMock,
}));

const encryptMock = vi.hoisted(() => vi.fn());
vi.mock("@/utils/sm2", () => ({
  getEncryptedLoginRequest: encryptMock,
}));

const menuApiMock = vi.hoisted(() => ({
  getUserMenus: vi.fn(),
  getAllUserMenus: vi.fn(),
  getUserPermissions: vi.fn(),
}));
vi.mock("@/lib/menuApi", () => menuApiMock);

import { useAuthStore, getTokenManager } from "./authStore";
import { useMenuStore } from "./menuStore";
import { STORAGE_KEYS } from "@/constants/storage";
import { resetMenuCache } from "@/services/cache/TTLMenuCache";

const fakeUser = { id: "u1", username: "tester", nickname: "测试" } as never;

function resetAuthState() {
  useAuthStore.setState({
    user: null,
    isAuthenticated: false,
    loading: false,
    menusLoaded: false,
    initialized: false,
  });
}

describe("authStore", () => {
  let consoleSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    sessionStorage.clear();
    localStorage.clear();
    resetMenuCache();
    vi.clearAllMocks();
    tokenManagerMock.getRefreshToken.mockResolvedValue(null);
    tokenManagerMock.initializeTokens.mockResolvedValue(undefined);
    tokenManagerMock.clearTokens.mockResolvedValue(undefined);
    tokenManagerMock.refreshToken.mockResolvedValue({
      accessToken: "a",
      refreshToken: "r",
      expiresIn: 3600,
    });
    resetAuthState();
    useMenuStore.setState({
      menus: [],
      allMenus: [],
      permissions: [],
      loading: false,
      lastFetchTime: null,
      error: null,
    });
    consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    consoleSpy.mockRestore();
    vi.useRealTimers();
  });

  it("login 成功:加密请求→TokenManager→认证状态→自动加载菜单", async () => {
    encryptMock.mockResolvedValue({
      username: "enc-user",
      password: "enc-pass",
      encryptedPassword: "enc-pwd",
    });
    apiPostMock.mockResolvedValue({
      data: {
        user: fakeUser,
        accessToken: "at",
        refreshToken: "rt",
        expiresIn: 7200,
      },
    });
    menuApiMock.getUserMenus.mockResolvedValue([]);
    menuApiMock.getAllUserMenus.mockResolvedValue([]);
    menuApiMock.getUserPermissions.mockResolvedValue(["system:user:list"]);

    await useAuthStore.getState().login({
      username: "tester",
      password: "secret",
    } as never);

    expect(encryptMock).toHaveBeenCalledWith("tester", "secret");
    expect(apiPostMock).toHaveBeenCalledWith("/system/auth/login", {
      username: "enc-user",
      password: "enc-pass",
      encryptedPassword: "enc-pwd",
      captcha: undefined,
      captchaId: undefined,
    });
    expect(tokenManagerMock.initializeTokens).toHaveBeenCalledWith("at", "rt", 7200);

    const state = useAuthStore.getState();
    expect(state.isAuthenticated).toBe(true);
    expect(state.user).toEqual(fakeUser);
    expect(state.loading).toBe(false);
    expect(state.menusLoaded).toBe(true);
    expect(state.initialized).toBe(true);
  });

  it("login 失败:loading 复位并向上抛出,menusLoaded 保持 false", async () => {
    encryptMock.mockResolvedValue({
      username: "u",
      password: "p",
      encryptedPassword: "e",
    });
    apiPostMock.mockRejectedValue(new Error("bad credentials"));

    await expect(
      useAuthStore.getState().login({ username: "x", password: "y" } as never)
    ).rejects.toThrow("bad credentials");

    const state = useAuthStore.getState();
    expect(state.loading).toBe(false);
    expect(state.isAuthenticated).toBe(false);
    expect(state.menusLoaded).toBe(false);
  });

  it("logout:清 token+菜单+存储痕迹并复位状态", async () => {
    useAuthStore.setState({ user: fakeUser, isAuthenticated: true, menusLoaded: true });
    sessionStorage.setItem(STORAGE_KEYS.LAST_PATH, "/system/user");
    sessionStorage.setItem("xingran_table_state_system_user_current", "5");
    localStorage.setItem("tabs-storage", "{}");
    useMenuStore.setState({ menus: [{ id: "m1" } as never], permissions: ["p"] });

    await useAuthStore.getState().logout();

    expect(tokenManagerMock.clearTokens).toHaveBeenCalledTimes(1);
    expect(useMenuStore.getState().menus).toEqual([]);
    expect(useMenuStore.getState().permissions).toEqual([]);
    expect(sessionStorage.getItem(STORAGE_KEYS.LAST_PATH)).toBeNull();
    expect(sessionStorage.getItem("xingran_table_state_system_user_current")).toBeNull();
    expect(localStorage.getItem("tabs-storage")).toBeNull();

    const state = useAuthStore.getState();
    expect(state.user).toBeNull();
    expect(state.isAuthenticated).toBe(false);
    expect(state.menusLoaded).toBe(false);
    expect(state.initialized).toBe(false);
  });

  it("logout 内部清理失败不阻塞状态复位", async () => {
    tokenManagerMock.clearTokens.mockRejectedValue(new Error("storage fail"));
    await useAuthStore.getState().logout();
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
    expect(consoleSpy).toHaveBeenCalled();
  });

  it("updateUser 合并字段;无 user 时 no-op", () => {
    useAuthStore.setState({ user: fakeUser });
    useAuthStore.getState().updateUser({ nickname: "新昵称" } as never);
    expect(useAuthStore.getState().user).toMatchObject({
      id: "u1",
      nickname: "新昵称",
    });

    useAuthStore.setState({ user: null });
    useAuthStore.getState().updateUser({ nickname: "x" } as never);
    expect(useAuthStore.getState().user).toBeNull();
  });

  it("loadMenusAfterLogin 成功置 menusLoaded;失败置 false 并抛出", async () => {
    menuApiMock.getUserMenus.mockResolvedValue([{}]);
    menuApiMock.getAllUserMenus.mockResolvedValue([{}]);
    menuApiMock.getUserPermissions.mockResolvedValue([]);
    await useAuthStore.getState().loadMenusAfterLogin();
    expect(useAuthStore.getState().menusLoaded).toBe(true);

    menuApiMock.getUserMenus.mockRejectedValue(new Error("menu fail"));
    await expect(useAuthStore.getState().loadMenusAfterLogin()).rejects.toThrow("menu fail");
    expect(useAuthStore.getState().menusLoaded).toBe(false);
  });

  it("getTokenManager 返回单例 TokenManager 实例", () => {
    expect(getTokenManager()).toBe(tokenManagerMock);
    expect(useAuthStore.getState().getTokenManager()).toBe(tokenManagerMock);
  });

  describe("initializeFromStorage", () => {
    it("已初始化时直接短路(防重复初始化)", async () => {
      useAuthStore.setState({ initialized: true });
      await useAuthStore.getState().initializeFromStorage();
      expect(tokenManagerMock.getRefreshToken).not.toHaveBeenCalled();
    });

    it("有 RefreshToken 且刷新成功 → 进入已认证状态(fake timers)", async () => {
      vi.useFakeTimers();
      tokenManagerMock.getRefreshToken.mockResolvedValue("encrypted-rt");

      const pending = useAuthStore.getState().initializeFromStorage();
      await vi.advanceTimersByTimeAsync(0);
      await pending;

      const state = useAuthStore.getState();
      expect(state.initialized).toBe(true);
      expect(state.isAuthenticated).toBe(true);
      expect(tokenManagerMock.refreshToken).toHaveBeenCalledTimes(1);
    });

    it("刷新失败 → 清理 token 并置未认证", async () => {
      tokenManagerMock.getRefreshToken.mockResolvedValue("expired-rt");
      tokenManagerMock.refreshToken.mockRejectedValue(new Error("refresh rejected"));

      await useAuthStore.getState().initializeFromStorage();

      expect(tokenManagerMock.clearTokens).toHaveBeenCalledTimes(1);
      const state = useAuthStore.getState();
      expect(state.initialized).toBe(true);
      expect(state.isAuthenticated).toBe(false);
      expect(state.user).toBeNull();
    });

    it("无 RefreshToken → 直接未认证", async () => {
      tokenManagerMock.getRefreshToken.mockResolvedValue(null);

      await useAuthStore.getState().initializeFromStorage();

      const state = useAuthStore.getState();
      expect(state.initialized).toBe(true);
      expect(state.isAuthenticated).toBe(false);
      expect(tokenManagerMock.refreshToken).not.toHaveBeenCalled();
    });

    it("getRefreshToken 抛 fatal 异常 → 兜底仍标记 initialized", async () => {
      tokenManagerMock.getRefreshToken.mockRejectedValue(new Error("storage broken"));

      await useAuthStore.getState().initializeFromStorage();

      const state = useAuthStore.getState();
      expect(state.initialized).toBe(true);
      expect(state.isAuthenticated).toBe(false);
      expect(consoleSpy).toHaveBeenCalled();
    });
  });
});
