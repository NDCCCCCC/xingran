/**
 * Phase 88 Batch344 — services/encryptionConfig 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/api", async () => {
  return {
    get: vi.fn(async (url: string) => ({
      data: { enabled: true, key: "k1", source: "src", url },
    })),
  };
});

import { get } from "@/lib/api";
import {
  getEncryptionConfig,
  getCachedEncryptionConfig,
  clearEncryptionConfigCache,
} from "../encryptionConfig";

describe("services/encryptionConfig", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    clearEncryptionConfigCache();
  });

  it("getEncryptionConfig 调用 /encryption-config", async () => {
    const r = await getEncryptionConfig();
    expect(get).toHaveBeenCalledWith("/system/auth/encryption-config");
    expect(r.enabled).toBe(true);
    expect(r.key).toBe("k1");
  });

  it("getCachedEncryptionConfig 第一次获取 → 调 API", async () => {
    await getCachedEncryptionConfig();
    expect(get).toHaveBeenCalled();
  });

  it("getCachedEncryptionConfig 缓存命中 → 不调 API", async () => {
    await getCachedEncryptionConfig();
    vi.mocked(get).mockClear();
    const r = await getCachedEncryptionConfig();
    expect(get).not.toHaveBeenCalled();
    expect(r.enabled).toBe(true);
  });

  it("clearEncryptionConfigCache 后 → 重新调 API", async () => {
    await getCachedEncryptionConfig();
    clearEncryptionConfigCache();
    vi.mocked(get).mockClear();
    await getCachedEncryptionConfig();
    expect(get).toHaveBeenCalled();
  });

  it("多次调用返回同一引用 (缓存)", async () => {
    const r1 = await getCachedEncryptionConfig();
    const r2 = await getCachedEncryptionConfig();
    expect(r1).toBe(r2);
  });
});
