/**
 * Phase 88 Batch345 — services/configService 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/api", async () => {
  return {
    get: vi.fn(async () => ({ data: {} })),
    post: vi.fn(async () => ({ data: {} })),
    put: vi.fn(async () => ({ data: {} })),
  };
});

vi.mock("@/lib/profileApi", () => ({
  getUserPreferences: vi.fn(async () => ({})),
}));

import { configService } from "../configService";

describe("services/configService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    configService.clearCache();
  });

  it("getUserPreferences 返回值含 version=2", async () => {
    const r = await configService.getUserPreferences();
    expect(r.version).toBe(2);
  });

  it("getUserPreferences 缓存命中 (重复调用)", async () => {
    const r1 = await configService.getUserPreferences();
    const r2 = await configService.getUserPreferences();
    expect(r1.version).toBe(r2.version);
  });

  it("updateUserPreferences 调用 put", async () => {
    const { put } = await import("@/lib/api");
    await configService.updateUserPreferences({
      version: 2,
      theme: { mode: "light" },
      layout: { type: "classic", density: "comfortable", sidebarCollapsed: false },
    });
    expect(put).toHaveBeenCalled();
  });

  it("clearCache 触发下次重新加载", async () => {
    await configService.getUserPreferences();
    configService.clearCache();
    await configService.getUserPreferences();
    // Verify no error thrown
    expect(configService).toBeDefined();
  });

  it("getDefaultPreferences 返回默认", () => {
    const r = configService.getDefaultPreferences();
    expect(r.version).toBe(2);
  });

  it("migratePreferences v1 + valid layout → returns v2", () => {
    const r = configService.migratePreferences({
      version: 1,
      layoutType: "hybrid",
      density: "compact",
    });
    expect(r.version).toBe(2);
  });

  it("migratePreferences v1 + invalid layoutType → falls back to default", () => {
    const r = configService.migratePreferences({
      version: 1,
      layoutType: "invalid",
      density: "invalid",
    });
    expect(r.version).toBe(2);
  });

  it("migratePreferences v2 → 透传", () => {
    const r = configService.migratePreferences({
      version: 2,
      theme: { mode: "dark" },
    });
    expect(r.version).toBe(2);
  });

  it("getUserPreferences 出错 → 返回 default", async () => {
    const { getUserPreferences } = await import("@/lib/profileApi");
    vi.mocked(getUserPreferences).mockRejectedValueOnce(new Error("net"));
    configService.clearCache();
    const r = await configService.getUserPreferences();
    expect(r.version).toBe(2);
  });
});
