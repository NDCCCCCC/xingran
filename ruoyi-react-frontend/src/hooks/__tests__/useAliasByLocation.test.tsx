/**
 * Phase 88 Batch398 — hooks/useAliasByLocation 测试
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

describe("hooks/useAliasByLocation", () => {
  it("useAliasByLocation 导出", async () => {
    const mod = await import("../useAliasByLocation");
    expect(typeof mod).toBe("object");
  });

  it("useAliasByLocation 是函数", async () => {
    const { useAliasByLocation } = await import("../useAliasByLocation");
    expect(typeof useAliasByLocation).toBe("function");
  });
});
