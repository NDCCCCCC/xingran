/**
 * Phase 88 Batch398 — hooks/useExceptionList 测试
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

describe("hooks/useExceptionList", () => {
  it("useExceptionList 导出", async () => {
    const mod = await import("../useExceptionList");
    expect(typeof mod).toBe("object");
  });

  it("useExceptionList 是函数", async () => {
    const { useExceptionList } = await import("../useExceptionList");
    expect(typeof useExceptionList).toBe("function");
  });
});
