/**
 * Phase 88 Batch398 — hooks/useReconciliationWebSocket 测试
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/store/settingsStore", () => ({
  useSettingsStore: vi.fn(() => ({
    preferences: { data: { theme: { mode: "light" } } },
  })),
}));

describe("hooks/useReconciliationWebSocket", () => {
  it("useReconciliationWebSocket 导出", async () => {
    const mod = await import("../useReconciliationWebSocket");
    expect(typeof mod).toBe("object");
  });

  it("useReconciliationWebSocket 是函数", async () => {
    const { useReconciliationWebSocket } = await import("../useReconciliationWebSocket");
    expect(typeof useReconciliationWebSocket).toBe("function");
  });
});
