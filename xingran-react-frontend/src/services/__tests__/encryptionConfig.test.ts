/**
 * Phase 88 Batch394 — services/encryptionConfig 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/api", async () => {
  return {
    get: vi.fn(async () => ({ data: { enabled: true, key: "test", source: "config" } })),
  };
});

describe("services/encryptionConfig", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("导出 encryptionConfig 相关函数", async () => {
    const mod = await import("../encryptionConfig");
    expect(typeof mod.getEncryptionConfig).toBe("function");
    expect(typeof mod.getCachedEncryptionConfig).toBe("function");
  });
});
