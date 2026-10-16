/**
 * Phase 88 Batch112 — utils/sm2 测试(65 stmts, 26.2% → 高)
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { createApiMock } from "@/test/utils/createApiMock";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const mockSM2 = {
  doEncrypt: vi.fn(() => "encrypted-hex"),
  doDecrypt: vi.fn(() => "decrypted"),
  doSignature: vi.fn(() => "sig-hex"),
  doVerifySignature: vi.fn(() => true),
  generateKeyPairHex: vi.fn(() => ({
    privateKey: "00priv",
    publicKey: "00pub",
  })),
  verifyPublicKey: vi.fn(() => true),
  comparePublicKeyHex: vi.fn(() => 0),
};

vi.mock("sm-crypto", () => ({
  sm2: mockSM2,
}));

import {
  fetchPublicKey,
  encryptWithSM2,
  decryptWithSM2,
  clearPublicKeyCache,
  getEncryptedLoginRequest,
  isSM2Available,
  generateSM2KeyPair,
  publicKeyToPEM,
} from "./sm2";

describe("utils/sm2", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    clearPublicKeyCache();
  });

  describe("fetchPublicKey", () => {
    it("缓存命中 → 不发请求", async () => {
      const api = createApiMock("/system/auth/public-key");
      api.endpoint.mockResolvedValueOnce({ code: 0, data: { publicKey: "abc" } } as any);
      await fetchPublicKey();
      api.endpoint.mockClear();
      const r2 = await fetchPublicKey();
      expect(r2).toBe("abc");
      expect(api.endpoint).not.toHaveBeenCalled();
    });

    it("forceRefresh → 重新请求", async () => {
      const api = createApiMock("/system/auth/public-key");
      api.endpoint.mockResolvedValue({ code: 0, data: { publicKey: "new" } } as any);
      const r = await fetchPublicKey(true);
      expect(r).toBe("new");
      expect(api.endpoint).toHaveBeenCalled();
    });

    it("code != 0 → 抛错", async () => {
      const api = createApiMock("/system/auth/public-key");
      api.endpoint.mockResolvedValueOnce({ code: 1, message: "fail" } as any);
      await expect(fetchPublicKey(true)).rejects.toThrow(/获取公钥失败/);
    });

    it("response.data 缺失 → 抛错", async () => {
      const api = createApiMock("/system/auth/public-key");
      api.endpoint.mockResolvedValueOnce({ code: 0, data: null } as any);
      await expect(fetchPublicKey(true)).rejects.toThrow(/获取公钥失败/);
    });
  });

  describe("encryptWithSM2 / decryptWithSM2", () => {
    it("encryptWithSM2 空密码 → 返回 ''", async () => {
      const r = await encryptWithSM2("", "00pub");
      expect(r).toBe("");
    });

    it("encryptWithSM2 正常 → 调用 doEncrypt + hexToBase64", async () => {
      const r = await encryptWithSM2("password", "00pub");
      expect(mockSM2.doEncrypt).toHaveBeenCalledWith("password", "00pub", 1);
      expect(typeof r).toBe("string");
      expect(r.length).toBeGreaterThan(0);
    });

    it("decryptWithSM2 空密文 → 返回 ''", async () => {
      const r = await decryptWithSM2("", "00priv");
      expect(r).toBe("");
    });

    it("decryptWithSM2 正常 → 调用 doDecrypt", async () => {
      const r = await decryptWithSM2("YWJj", "00priv");
      expect(mockSM2.doDecrypt).toHaveBeenCalledWith("616263", "00priv", 1);
      expect(r).toBe("decrypted");
    });
  });

  describe("clearPublicKeyCache", () => {
    it("清除后下次重新请求", async () => {
      const api = createApiMock("/system/auth/public-key");
      api.endpoint.mockResolvedValue({ code: 0, data: { publicKey: "x" } } as any);
      await fetchPublicKey();
      clearPublicKeyCache();
      api.endpoint.mockClear();
      await fetchPublicKey();
      expect(api.endpoint).toHaveBeenCalled();
    });
  });

  describe("getEncryptedLoginRequest", () => {
    it("成功 → encryptedPassword=true", async () => {
      const api = createApiMock("/system/auth/public-key");
      api.endpoint.mockResolvedValueOnce({ code: 0, data: { publicKey: "00pub" } } as any);
      const r = await getEncryptedLoginRequest("admin", "pass");
      expect(r).toEqual({
        username: "admin",
        password: expect.any(String),
        encryptedPassword: true,
      });
    });

    it("失败 → encryptedPassword=false (开发环境)", async () => {
      const api = createApiMock("/system/auth/public-key");
      api.endpoint.mockRejectedValueOnce(new Error("net"));
      const r = await getEncryptedLoginRequest("admin", "pass");
      expect(r.encryptedPassword).toBe(false);
    });
  });

  describe("isSM2Available", () => {
    it("成功 → true", async () => {
      const r = await isSM2Available();
      expect(r).toBe(true);
    });
  });

  describe("generateSM2KeyPair", () => {
    it("返回密钥对", async () => {
      const r = await generateSM2KeyPair();
      expect(r).toEqual({ privateKey: "00priv", publicKey: "00pub" });
    });
  });

  describe("publicKeyToPEM", () => {
    it("生成 PEM 格式", () => {
      const pem = publicKeyToPEM("48656c6c6f");
      expect(pem).toContain("-----BEGIN PUBLIC KEY-----");
      expect(pem).toContain("-----END OF PUBLIC KEY-----");
    });

    it("空 hex → 仅含 BEGIN/END 头尾", () => {
      const pem = publicKeyToPEM("");
      expect(pem).toContain("BEGIN PUBLIC KEY");
      expect(pem).toContain("END OF PUBLIC KEY");
    });
  });
});
