/**
 * TokenManager 测试（D-09: vi.useFakeTimers 驱动定时刷新循环）
 *
 * - @/lib/api 的 post 用 vi.mock 拦截（T-83-02: 不发真实网络请求）
 * - storage 用轻量 FakeStorage（内存实现 SecureTokenStorage 接口）
 * - fake timers 同时接管 Date.now，锁超时分支可确定性触发（T-83-02-03 防 flaky）
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { SecureTokenStorage, TokenMeta } from "./SecureTokenStorage";
import { TokenRefreshError } from "./SecureTokenStorage";

const mockPost = vi.hoisted(() => vi.fn());

vi.mock("@/lib/api", () => ({
  post: mockPost,
}));

import { TokenManager } from "./TokenManager";
import type { TokenManagerConfig } from "./TokenManager";

class FakeStorage implements SecureTokenStorage {
  accessToken: string | null = null;
  refreshToken: string | null = null;
  tokenMeta: TokenMeta | null = null;

  setAccessToken(token: string): void {
    this.accessToken = token;
  }
  getAccessToken(): string | null {
    return this.accessToken;
  }
  async setRefreshToken(token: string): Promise<void> {
    this.refreshToken = token;
  }
  async getRefreshToken(): Promise<string | null> {
    return this.refreshToken;
  }
  setTokenMeta(meta: TokenMeta): void {
    this.tokenMeta = meta;
  }
  getTokenMeta(): TokenMeta | null {
    return this.tokenMeta;
  }
  async clear(): Promise<void> {
    this.accessToken = null;
    this.refreshToken = null;
    this.tokenMeta = null;
  }
  isAccessTokenExpiringWithin(seconds: number): boolean {
    if (!this.tokenMeta || !this.tokenMeta.expiresAt) return false;
    return this.tokenMeta.expiresAt - Date.now() <= seconds * 1000;
  }
}

function createManager(overrides: Partial<TokenManagerConfig> = {}) {
  storage = new FakeStorage();
  manager = new TokenManager(storage, {
    refreshEndpoint: "/system/auth/refresh",
    refreshBeforeSeconds: 30,
    refreshTimeout: 10_000,
    ...overrides,
  });
  return manager;
}

let storage: FakeStorage;
let manager: TokenManager;

describe("TokenManager", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mockPost.mockReset();
    vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("TokenRefreshError 携带 name 与错误码", () => {
    const error = new TokenRefreshError("expired", "INVALID_TOKEN");
    expect(error).toBeInstanceOf(Error);
    expect(error.name).toBe("TokenRefreshError");
    expect(error.message).toBe("expired");
    expect(error.code).toBe("INVALID_TOKEN");
  });

  it("initializeTokens 写入三类状态并调度刷新定时器", async () => {
    createManager();
    await manager.initializeTokens("acc-1", "ref-1", 60);

    expect(storage.accessToken).toBe("acc-1");
    expect(storage.refreshToken).toBe("ref-1");
    expect(storage.tokenMeta?.expiresIn).toBe(60);
    expect(manager.isAuthenticated()).toBe(true);
    expect(manager.getTokenRemainingTime()).toBe(60);
    // 定时器未到期前不应发起刷新
    expect(mockPost).not.toHaveBeenCalled();
  });

  it("无 AccessToken 时 getAccessToken 抛 INVALID_TOKEN", async () => {
    createManager();
    await expect(manager.getAccessToken()).rejects.toMatchObject({
      code: "INVALID_TOKEN",
    });
    expect(mockPost).not.toHaveBeenCalled();
  });

  it("Token 远未过期时 getAccessToken 直接返回（不刷新）", async () => {
    createManager();
    storage.accessToken = "fresh-acc";
    storage.tokenMeta = {
      expiresAt: Date.now() + 3_600_000,
      issuedAt: Date.now(),
      expiresIn: 3600,
    };

    await expect(manager.getAccessToken()).resolves.toBe("fresh-acc");
    expect(mockPost).not.toHaveBeenCalled();
  });

  it("Token 即将过期时 getAccessToken 自动刷新并返回新 Token", async () => {
    createManager();
    storage.accessToken = "old-acc";
    storage.refreshToken = "old-ref";
    // 距过期 5s < refreshBeforeSeconds 30s → 触发刷新
    storage.tokenMeta = {
      expiresAt: Date.now() + 5_000,
      issuedAt: Date.now(),
      expiresIn: 5,
    };
    mockPost.mockResolvedValueOnce({
      data: { accessToken: "new-acc", refreshToken: "new-ref", expiresIn: 3600 },
    });

    await expect(manager.getAccessToken()).resolves.toBe("new-acc");
    expect(mockPost).toHaveBeenCalledTimes(1);
    expect(mockPost).toHaveBeenCalledWith("/system/auth/refresh", {
      refreshToken: "old-ref",
    });
    expect(storage.refreshToken).toBe("new-ref");
  });

  it("定时器在过期前 refreshBeforeSeconds 自动触发刷新（fake timers）", async () => {
    createManager();
    mockPost.mockResolvedValueOnce({
      data: { accessToken: "acc-2", refreshToken: "ref-2", expiresIn: 3600 },
    });

    await manager.initializeTokens("acc-1", "ref-1", 60); // 刷新点 = +30s
    expect(mockPost).not.toHaveBeenCalled();

    await vi.advanceTimersByTimeAsync(31_000);

    expect(mockPost).toHaveBeenCalledTimes(1);
    expect(storage.accessToken).toBe("acc-2");
    await expect(manager.getRefreshToken()).resolves.toBe("ref-2");
  });

  it("并发刷新只发一次请求（刷新锁 single-flight, T-83-02-04）", async () => {
    createManager();
    storage.refreshToken = "seed-ref";

    let resolvePost!: (value: unknown) => void;
    mockPost.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolvePost = resolve;
        })
    );

    const p1 = manager.refreshToken();
    const p2 = manager.refreshToken();
    const p3 = manager.refreshToken();

    await vi.advanceTimersByTimeAsync(0); // 冲刷 doRefresh 首个 await 的微任务
    expect(mockPost).toHaveBeenCalledTimes(1);

    resolvePost({
      data: { accessToken: "a-once", refreshToken: "r-once", expiresIn: 60 },
    });
    const results = await Promise.all([p1, p2, p3]);

    expect(results[0]).toEqual(results[1]);
    expect(results[0]).toEqual(results[2]);
    expect(results[0]).toMatchObject({ accessToken: "a-once" });
    expect(mockPost).toHaveBeenCalledTimes(1);
  });

  it("刷新锁超过 refreshTimeout 后放行新刷新（陈旧锁清理）", async () => {
    createManager({ refreshTimeout: 100 });
    storage.refreshToken = "seed-ref";

    mockPost.mockImplementationOnce(() => new Promise(() => {})); // 永不结算
    void manager.refreshToken(); // 占锁
    await vi.advanceTimersByTimeAsync(0); // 冲刷微任务，让首个 doRefresh 真正发起 post
    expect(mockPost).toHaveBeenCalledTimes(1);

    vi.advanceTimersByTime(200); // 锁时间戳 +200ms > 100ms 超时

    mockPost.mockImplementationOnce(() => new Promise(() => {}));
    void manager.refreshToken(); // 陈旧锁被清除 → 新请求
    await vi.advanceTimersByTimeAsync(0);
    expect(mockPost).toHaveBeenCalledTimes(2);
  });

  it("401 刷新失败抛 INVALID_TOKEN，且失败后锁被释放可重试", async () => {
    createManager();
    storage.refreshToken = "expired-ref";
    mockPost.mockRejectedValueOnce({ response: { status: 401 } });

    await expect(manager.refreshToken()).rejects.toMatchObject({
      code: "INVALID_TOKEN",
      message: "Refresh token expired",
    });

    // finally 分支清除锁 → 第二次调用发起新请求并成功
    mockPost.mockResolvedValueOnce({
      data: { accessToken: "retry-acc", refreshToken: "retry-ref", expiresIn: 60 },
    });
    await expect(manager.refreshToken()).resolves.toMatchObject({
      accessToken: "retry-acc",
    });
  });

  it("网络错误映射 NETWORK_ERROR，其他错误映射 SERVER_ERROR", async () => {
    createManager();
    storage.refreshToken = "ref";

    mockPost.mockRejectedValueOnce({ code: "NETWORK_ERROR" });
    await expect(manager.refreshToken()).rejects.toMatchObject({ code: "NETWORK_ERROR" });

    mockPost.mockRejectedValueOnce(new Error("boom"));
    await expect(manager.refreshToken()).rejects.toMatchObject({ code: "SERVER_ERROR" });
    expect(console.error).toHaveBeenCalled();
  });

  it("无 RefreshToken 时刷新抛 INVALID_TOKEN 且不发起请求", async () => {
    createManager();
    storage.refreshToken = null;

    await expect(manager.refreshToken()).rejects.toMatchObject({ code: "INVALID_TOKEN" });
    expect(mockPost).not.toHaveBeenCalled();
  });

  it("clearTokens 清存储、停定时器，之后不再触发刷新", async () => {
    createManager();
    await manager.initializeTokens("acc", "ref", 3600); // 定时器 +3570s
    expect(manager.isAuthenticated()).toBe(true);

    await manager.clearTokens();

    expect(manager.isAuthenticated()).toBe(false);
    expect(storage.accessToken).toBeNull();
    expect(storage.refreshToken).toBeNull();

    await vi.advanceTimersByTimeAsync(4_000_000);
    expect(mockPost).not.toHaveBeenCalled();
  });

  it("无 meta 时 getTokenRemainingTime 返回 0", () => {
    createManager();
    expect(manager.getTokenRemainingTime()).toBe(0);
  });
});
