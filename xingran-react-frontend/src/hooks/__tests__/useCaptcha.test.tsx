/**
 * Phase 88 Batch398 — hooks/useCaptcha 测试
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

describe("hooks/useCaptcha", () => {
  it("useCaptcha 导出", async () => {
    const mod = await import("../useCaptcha");
    expect(typeof mod).toBe("object");
  });

  it("useCaptcha 是函数", async () => {
    const { useCaptcha } = await import("../useCaptcha");
    expect(typeof useCaptcha).toBe("function");
  });
});
