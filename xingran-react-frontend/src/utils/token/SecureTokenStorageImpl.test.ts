/**
 * SecureTokenStorageImpl 测试
 *
 * - 走真实 SM4 加解密（D-08，依赖链 @/utils/sm4 → sm-crypto 真实调用）
 * - 仅 mock @/lib/api（sm4.ts 的网络依赖 fetchSM4KeyForPassword 用，本测试不触达）
 * - 测试向量均为假 token（T-83-02-01：不引入真实凭证）
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mockGet = vi.hoisted(() => vi.fn());

vi.mock("@/lib/api", () => ({
  get: mockGet,
}));

import { SecureTokenStorageImpl } from "./SecureTokenStorageImpl";
import type { TokenMeta } from "./SecureTokenStorage";

describe("SecureTokenStorageImpl（真实 SM4-CBC）", () => {
  let storage: SecureTokenStorageImpl;

  beforeEach(() => {
    sessionStorage.clear();
    localStorage.clear();
    vi.spyOn(console, "error").mockImplementation(() => {});
    storage = new SecureTokenStorageImpl();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("AccessToken 存内存：set 后可读，不落 sessionStorage", () => {
    expect(storage.getAccessToken()).toBeNull();
    storage.setAccessToken("fake-access-token");
    expect(storage.getAccessToken()).toBe("fake-access-token");
    expect(sessionStorage.length).toBe(0);
  });

  it("RefreshToken 加密写入 sessionStorage（密文不含明文）并可解密读回", async () => {
    const plaintext = "fake-refresh-token-测试向量";
    await storage.setRefreshToken(plaintext);

    const raw = sessionStorage.getItem("rt");
    expect(raw).toBeTruthy();
    expect(raw).not.toContain(plaintext);
    expect(raw).not.toContain(plaintext.slice(0, 10));

    await expect(storage.getRefreshToken()).resolves.toBe(plaintext);
  });

  it("未存 RefreshToken 时返回 null", async () => {
    await expect(storage.getRefreshToken()).resolves.toBeNull();
  });

  it("sessionStorage 损坏数据时回退 null 并清除脏数据", async () => {
    sessionStorage.setItem("rt", "###not-base64###");
    await expect(storage.getRefreshToken()).resolves.toBeNull();
    expect(sessionStorage.getItem("rt")).toBeNull();
  });

  it("TokenMeta 内存优先 + sessionStorage 持久化（页面刷新恢复）", () => {
    const meta: TokenMeta = {
      expiresAt: Date.now() + 60_000,
      issuedAt: Date.now(),
      expiresIn: 60,
    };
    storage.setTokenMeta(meta);
    expect(storage.getTokenMeta()).toEqual(meta);
    expect(sessionStorage.getItem("tm")).toContain(String(meta.expiresAt));

    // 新实例（模拟页面刷新）从 sessionStorage 恢复
    const revived = new SecureTokenStorageImpl();
    expect(revived.getTokenMeta()).toEqual(meta);
  });

  it("TokenMeta 损坏 JSON 时返回 null（静默兜底）", () => {
    sessionStorage.setItem("tm", "{broken-json");
    expect(new SecureTokenStorageImpl().getTokenMeta()).toBeNull();
  });

  it("clear 清空内存与 sessionStorage", async () => {
    storage.setAccessToken("acc");
    await storage.setRefreshToken("ref");
    storage.setTokenMeta({ expiresAt: 1, issuedAt: 1, expiresIn: 1 });

    await storage.clear();

    expect(storage.getAccessToken()).toBeNull();
    await expect(storage.getRefreshToken()).resolves.toBeNull();
    expect(storage.getTokenMeta()).toBeNull();
    expect(sessionStorage.getItem("rt")).toBeNull();
    expect(sessionStorage.getItem("tm")).toBeNull();
  });

  describe("isAccessTokenExpiringWithin", () => {
    it("无 meta 时返回 false", () => {
      expect(storage.isAccessTokenExpiringWithin(30)).toBe(false);
    });

    it("距过期 60s、阈值 30s 时不触发", () => {
      storage.setTokenMeta({
        expiresAt: Date.now() + 60_000,
        issuedAt: Date.now(),
        expiresIn: 60,
      });
      expect(storage.isAccessTokenExpiringWithin(30)).toBe(false);
    });

    it("距过期 10s、阈值 30s 时触发", () => {
      storage.setTokenMeta({
        expiresAt: Date.now() + 10_000,
        issuedAt: Date.now(),
        expiresIn: 10,
      });
      expect(storage.isAccessTokenExpiringWithin(30)).toBe(true);
    });

    it("已过期时触发", () => {
      storage.setTokenMeta({
        expiresAt: Date.now() - 1_000,
        issuedAt: Date.now() - 10_000,
        expiresIn: 10,
      });
      expect(storage.isAccessTokenExpiringWithin(30)).toBe(true);
    });
  });
});
