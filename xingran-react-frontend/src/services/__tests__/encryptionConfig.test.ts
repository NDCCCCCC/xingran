/**
 * Phase 88 Batch207 — services/encryptionConfig 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/api", async () => {
  return {
    get: vi.fn(async () => ({
      data: { enabled: true, key: "test-key", source: "backend" },
    })),
  };
});

import * as api from "@/lib/api";
import {
  getEncryptionConfig,
  getCachedEncryptionConfig,
  clearEncryptionConfigCache,
} from "../encryptionConfig";

describe("services/encryptionConfig", () => {
  beforeEach(() => {
    clearEncryptionConfigCache();
    vi.clearAllMocks();
  });

  it("getEncryptionConfig 调用 get", async () => {
    const r = await getEncryptionConfig();
    expect(r.enabled).toBe(true);
    expect(api.get).toHaveBeenCalledWith("/system/auth/encryption-config");
  });

  it("getCachedEncryptionConfig 首次拉取", async () => {
    const r = await getCachedEncryptionConfig();
    expect(r.enabled).toBe(true);
  });

  it("getCachedEncryptionConfig 缓存命中", async () => {
    await getCachedEncryptionConfig();
    await getCachedEncryptionConfig();
    expect(api.get).toHaveBeenCalledTimes(1);
  });

  it("clearEncryptionConfigCache 清缓存", async () => {
    await getCachedEncryptionConfig();
    clearEncryptionConfigCache();
    await getCachedEncryptionConfig();
    expect(api.get).toHaveBeenCalledTimes(2);
  });
});
