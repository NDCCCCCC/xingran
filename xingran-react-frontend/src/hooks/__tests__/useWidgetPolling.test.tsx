/**
 * Phase 88 Batch398 — hooks/useWidgetPolling 测试
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/store/dashboardStore", () => {
  // Per-field selector (Phase 116 DASH-03) — pass selector through state.
  const state = {
    getCachedWidgetData: vi.fn(),
    cacheWidgetData: vi.fn(),
  };
  return {
    useDashboardStore: vi.fn((selector?: (s: typeof state) => unknown) =>
      typeof selector === "function" ? selector(state) : state
    ),
  };
});

describe("hooks/useWidgetPolling", () => {
  it("useWidgetPolling 导出", async () => {
    const mod = await import("../useWidgetPolling");
    expect(typeof mod).toBe("object");
  });

  it("useWidgetPolling 是函数", async () => {
    const { useWidgetPolling } = await import("../useWidgetPolling");
    expect(typeof useWidgetPolling).toBe("function");
  });
});
