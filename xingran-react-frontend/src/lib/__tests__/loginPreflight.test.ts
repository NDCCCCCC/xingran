/**
 * Phase 88 Batch304 — lib/loginPreflight 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  const actual = await createApiTestingModule();
  return { ...actual, refreshEncryptionConfig: vi.fn(async () => true) };
});

vi.mock("@/services/captcha", () => ({
  getCaptchaConfig: vi.fn(async () => ({ enabled: true })),
}));

vi.mock("@/utils/sm2", () => ({
  clearPublicKeyCache: vi.fn(),
  fetchPublicKey: vi.fn(async () => "fake-public-key"),
}));

import { refreshEncryptionConfig } from "@/lib/api";
import { getCaptchaConfig } from "@/services/captcha";
import { fetchPublicKey, clearPublicKeyCache } from "@/utils/sm2";
import { submitLoginPreflight } from "../loginPreflight";

describe("lib/loginPreflight", () => {
  beforeEach(() => {
    vi.mocked(refreshEncryptionConfig).mockReset();
    vi.mocked(getCaptchaConfig).mockReset();
    vi.mocked(fetchPublicKey).mockReset();
    vi.mocked(clearPublicKeyCache).mockReset();
  });

  it("三项成功 → ok=true + captchaEnabled", async () => {
    vi.mocked(refreshEncryptionConfig).mockResolvedValue(true as any);
    vi.mocked(getCaptchaConfig).mockResolvedValue({ enabled: true } as any);
    vi.mocked(fetchPublicKey).mockResolvedValue("key" as any);

    const r = await submitLoginPreflight();
    expect(r.ok).toBe(true);
    if (r.ok) expect(r.captchaEnabled).toBe(true);
  });

  it("captcha=false 也 ok=true", async () => {
    vi.mocked(refreshEncryptionConfig).mockResolvedValue(true as any);
    vi.mocked(getCaptchaConfig).mockResolvedValue({ enabled: false } as any);
    vi.mocked(fetchPublicKey).mockResolvedValue("key" as any);

    const r = await submitLoginPreflight();
    expect(r.ok).toBe(true);
    if (r.ok) expect(r.captchaEnabled).toBe(false);
  });

  it("encryption 失败 → ok=false", async () => {
    vi.mocked(refreshEncryptionConfig).mockResolvedValue(false as any);
    vi.mocked(getCaptchaConfig).mockResolvedValue({ enabled: true } as any);
    vi.mocked(fetchPublicKey).mockResolvedValue("key" as any);

    const r = await submitLoginPreflight();
    expect(r.ok).toBe(false);
  });

  it("publicKey 失败 → ok=false", async () => {
    vi.mocked(refreshEncryptionConfig).mockResolvedValue(true as any);
    vi.mocked(getCaptchaConfig).mockResolvedValue({ enabled: true } as any);
    vi.mocked(fetchPublicKey).mockRejectedValue(new Error("net"));

    const r = await submitLoginPreflight();
    expect(r.ok).toBe(false);
    if (!r.ok) expect(r.friendlyMessage).toContain("失败");
  });

  it("captcha 失败 → ok=false", async () => {
    vi.mocked(refreshEncryptionConfig).mockResolvedValue(true as any);
    vi.mocked(getCaptchaConfig).mockRejectedValue(new Error("cap"));
    vi.mocked(fetchPublicKey).mockResolvedValue("key" as any);

    const r = await submitLoginPreflight();
    expect(r.ok).toBe(false);
  });

  it("成功时 clearPublicKeyCache 被调用", async () => {
    vi.mocked(refreshEncryptionConfig).mockResolvedValue(true as any);
    vi.mocked(getCaptchaConfig).mockResolvedValue({ enabled: true } as any);
    vi.mocked(fetchPublicKey).mockResolvedValue("key" as any);

    await submitLoginPreflight();
    expect(clearPublicKeyCache).toHaveBeenCalled();
  });

  it("fetchPublicKey 收到 true 参数", async () => {
    vi.mocked(refreshEncryptionConfig).mockResolvedValue(true as any);
    vi.mocked(getCaptchaConfig).mockResolvedValue({ enabled: true } as any);
    vi.mocked(fetchPublicKey).mockResolvedValue("key" as any);

    await submitLoginPreflight();
    expect(fetchPublicKey).toHaveBeenCalledWith(true);
  });
});
