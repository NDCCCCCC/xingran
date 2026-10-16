/**
 * Phase 88 Batch392 — store/layoutStore 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/constants/storage", () => ({
  ZUSTAND_STORAGE_KEYS: {},
}));

vi.mock("@/types/config", () => ({
  defaultLayoutConfiguration: {},
}));

vi.mock("@/types/layout", () => ({}));

describe("store/layoutStore", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("layoutStore 有导出", async () => {
    const mod = await import("../layoutStore");
    expect(typeof mod).toBe("object");
  });

  it("useLayoutStore 是函数", async () => {
    const { useLayoutStore } = await import("../layoutStore");
    expect(typeof useLayoutStore).toBe("function");
  });

  it("初始状态包含 layout 相关字段", async () => {
    const { useLayoutStore } = await import("../layoutStore");
    const state = useLayoutStore.getState();
    expect(typeof state.currentLayout).toBe("string");
    expect(typeof state.sidebarCollapsed).toBe("boolean");
    expect(typeof state.density).toBe("string");
  });

  it("有 toggleSidebar / setSidebarCollapsed / setDensity / setLayout 方法", async () => {
    const { useLayoutStore } = await import("../layoutStore");
    const state = useLayoutStore.getState();
    expect(typeof state.toggleSidebar).toBe("function");
    expect(typeof state.setSidebarCollapsed).toBe("function");
    expect(typeof state.setDensity).toBe("function");
    expect(typeof state.setLayout).toBe("function");
  });

  it("syncFromSettings 是函数", async () => {
    const { useLayoutStore } = await import("../layoutStore");
    const state = useLayoutStore.getState();
    expect(typeof state.syncFromSettings).toBe("function");
  });

  it("saveState 是函数", async () => {
    const { useLayoutStore } = await import("../layoutStore");
    const state = useLayoutStore.getState();
    expect(typeof state.saveState).toBe("function");
  });

  it("applyToDOM 是函数", async () => {
    const { useLayoutStore } = await import("../layoutStore");
    const state = useLayoutStore.getState();
    expect(typeof state.applyToDOM).toBe("function");
  });
});
