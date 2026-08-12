import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const {
  mockRefreshEncryptionConfig,
  mockGetCaptchaConfig,
  mockFetchPublicKey,
  mockClearPublicKeyCache,
} = vi.hoisted(() => ({
  mockRefreshEncryptionConfig: vi.fn<() => Promise<boolean>>(),
  mockGetCaptchaConfig: vi.fn(),
  mockFetchPublicKey: vi.fn(),
  mockClearPublicKeyCache: vi.fn(),
}));

vi.mock("@/lib/api", () => ({
  refreshEncryptionConfig: mockRefreshEncryptionConfig,
}));

vi.mock("@/services/captcha", () => ({
  getCaptchaConfig: mockGetCaptchaConfig,
}));

vi.mock("@/utils/sm2", () => ({
  fetchPublicKey: mockFetchPublicKey,
  clearPublicKeyCache: mockClearPublicKeyCache,
}));

import { submitLoginPreflight } from "@/lib/loginPreflight";

describe("submitLoginPreflight", () => {
  beforeEach(() => {
    vi.spyOn(console, "error").mockImplementation(() => {});
    mockRefreshEncryptionConfig.mockReset();
    mockGetCaptchaConfig.mockReset();
    mockFetchPublicKey.mockReset();
    mockClearPublicKeyCache.mockReset();

    mockRefreshEncryptionConfig.mockResolvedValue(true);
    mockGetCaptchaConfig.mockResolvedValue({ enabled: "disabled" });
    mockFetchPublicKey.mockResolvedValue("04ABCD");
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("返回本次刷新得到的验证码类型", async () => {
    mockGetCaptchaConfig.mockResolvedValue({ enabled: "slider" });

    const result = await submitLoginPreflight();

    expect(result).toEqual({ ok: true, captchaEnabled: "slider" });
  });

  it("并发刷新加密开关、公钥和验证码配置", async () => {
    let resolveEncryption!: (value: boolean) => void;
    mockRefreshEncryptionConfig.mockImplementation(
      () => new Promise<boolean>((resolve) => {
        resolveEncryption = resolve;
      })
    );

    const pending = submitLoginPreflight();
    await Promise.resolve();

    expect(mockRefreshEncryptionConfig).toHaveBeenCalledTimes(1);
    expect(mockFetchPublicKey).toHaveBeenCalledWith(true);
    expect(mockGetCaptchaConfig).toHaveBeenCalledTimes(1);

    resolveEncryption(true);
    await pending;
  });

  it("加密配置刷新返回 false 时阻止继续登录", async () => {
    mockRefreshEncryptionConfig.mockResolvedValue(false);

    const result = await submitLoginPreflight();

    expect(result).toEqual({
      ok: false,
      friendlyMessage: "登录安全配置已过期，自动更新失败，请检查网络后重试",
    });
  });

  it("加密配置刷新抛错时返回友好提示", async () => {
    mockRefreshEncryptionConfig.mockRejectedValue(new Error("network down"));

    const result = await submitLoginPreflight();

    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.friendlyMessage).not.toContain("network down");
    }
  });

  it("验证码配置刷新失败时返回友好提示", async () => {
    mockGetCaptchaConfig.mockRejectedValue(new Error("captcha service 500"));

    const result = await submitLoginPreflight();

    expect(result.ok).toBe(false);
  });

  it("强制清除并刷新 SM2 公钥", async () => {
    await submitLoginPreflight();

    expect(mockClearPublicKeyCache).toHaveBeenCalledTimes(1);
    expect(mockFetchPublicKey).toHaveBeenCalledWith(true);
  });

  it("SM2 公钥刷新失败时返回友好提示", async () => {
    mockFetchPublicKey.mockRejectedValue(new Error("public key rotated"));

    const result = await submitLoginPreflight();

    expect(result.ok).toBe(false);
  });

  it("刷新超过 5 秒时停止等待并返回友好提示", async () => {
    mockRefreshEncryptionConfig.mockImplementation(() => new Promise(() => {}));

    vi.useFakeTimers();
    try {
      const pending = submitLoginPreflight();
      await vi.advanceTimersByTimeAsync(5000);

      expect(await pending).toEqual({
        ok: false,
        friendlyMessage: "登录安全配置已过期，自动更新失败，请检查网络后重试",
      });
    } finally {
      vi.useRealTimers();
    }
  });
});
