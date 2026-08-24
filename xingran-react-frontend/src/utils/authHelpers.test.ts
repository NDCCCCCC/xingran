import { beforeEach, describe, expect, it, vi } from "vitest";

const {
  mockGetTokenManager,
  mockTokenGetAccessToken,
  mockGetCachedEncryptionConfig,
  mockClearEncryptionConfigCache,
} = vi.hoisted(() => ({
  mockGetTokenManager: vi.fn(),
  mockTokenGetAccessToken: vi.fn(),
  mockGetCachedEncryptionConfig: vi.fn(),
  mockClearEncryptionConfigCache: vi.fn(),
}));

vi.mock("@/store/authStore", () => ({
  getTokenManager: mockGetTokenManager,
}));

vi.mock("@/services/encryptionConfig", () => ({
  getCachedEncryptionConfig: mockGetCachedEncryptionConfig,
  clearEncryptionConfigCache: mockClearEncryptionConfigCache,
}));

import {
  getAccessToken,
  getAuthHeaders,
  getEncryptionConfigStatus,
  refreshEncryptionConfig,
  withAuth,
} from "./authHelpers";

function mockTokenManagerResolve(token: string) {
  mockTokenGetAccessToken.mockResolvedValueOnce(token);
  mockGetTokenManager.mockReset();
  mockGetTokenManager.mockReturnValue({ getAccessToken: mockTokenGetAccessToken });
}

describe("authHelpers（token/加密配置辅助）", () => {
  beforeEach(() => {
    mockTokenGetAccessToken.mockReset();
    mockGetCachedEncryptionConfig.mockReset();
    mockClearEncryptionConfigCache.mockReset();
    vi.spyOn(console, "warn").mockImplementation(() => {});
  });

  it("getAuthHeaders 携带 Bearer token", async () => {
    mockTokenManagerResolve("fake-jwt");
    await expect(getAuthHeaders()).resolves.toEqual({ Authorization: "Bearer fake-jwt" });
  });

  it("token 为空字符串时返回空 headers（不发送空 Authorization）", async () => {
    mockTokenManagerResolve("");
    await expect(getAuthHeaders()).resolves.toEqual({});
  });

  it("getAccessToken 透传 TokenManager 结果", async () => {
    mockTokenManagerResolve("passthrough-token");
    await expect(getAccessToken()).resolves.toBe("passthrough-token");
  });

  it("withAuth 合并既有 headers 与认证头", async () => {
    mockTokenManagerResolve("tok");
    const [url, options] = await withAuth("/api/thing", {
      method: "POST",
      headers: { "X-Custom": "1" },
    });
    expect(url).toBe("/api/thing");
    expect(options.method).toBe("POST");
    expect(options.headers).toEqual({
      "X-Custom": "1",
      Authorization: "Bearer tok",
    });
  });

  it("withAuth 无 options 时仍带认证头", async () => {
    mockTokenManagerResolve("tok");
    const [, options] = await withAuth("/api/thing");
    expect(options.headers).toEqual({ Authorization: "Bearer tok" });
  });

  it("refreshEncryptionConfig 先清缓存再重新拉取", async () => {
    const fresh = { enabled: true, excludePaths: [] };
    mockGetCachedEncryptionConfig.mockResolvedValueOnce(fresh);
    await expect(refreshEncryptionConfig()).resolves.toBe(fresh);
    expect(mockClearEncryptionConfigCache).toHaveBeenCalledTimes(1);
    expect(mockGetCachedEncryptionConfig).toHaveBeenCalledTimes(1);
  });

  it("getEncryptionConfigStatus 返回 enabled 开关", async () => {
    mockGetCachedEncryptionConfig.mockResolvedValueOnce({ enabled: true });
    await expect(getEncryptionConfigStatus()).resolves.toBe(true);
    mockGetCachedEncryptionConfig.mockResolvedValueOnce({ enabled: false });
    await expect(getEncryptionConfigStatus()).resolves.toBe(false);
  });

  it("getEncryptionConfigStatus 失败时 fail-safe 返回 true", async () => {
    mockGetCachedEncryptionConfig.mockRejectedValueOnce(new Error("network"));
    await expect(getEncryptionConfigStatus()).resolves.toBe(true);
    expect(console.warn).toHaveBeenCalled();
  });
});
