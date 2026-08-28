/**
 * Phase 87 — ad(SyncMonitor) 模块导入断言
 */
import { describe, it, expect } from "vitest";

describe("ad page modules", () => {
  it("SyncMonitor imports", { timeout: 30000 }, async () => {
    const m = await import("../SyncMonitor");
    expect(m.default ?? m).toBeDefined();
  });
});
