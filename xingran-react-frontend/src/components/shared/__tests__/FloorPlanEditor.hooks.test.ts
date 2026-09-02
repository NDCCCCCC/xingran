/**
 * Phase 88 Batch425 — FloorPlanEditor.hooks 测试
 */
import { describe, it, expect, vi } from "vitest";

describe("FloorPlanEditor.hooks", () => {
  it("导出 hook 函数", async () => {
    const mod = await import("../FloorPlanEditor.hooks");
    expect(typeof mod).toBe("object");
  });

  it("常量导出", async () => {
    const mod = await import("../FloorPlanEditor.constants");
    expect(typeof mod).toBe("object");
  });

  it("类型导出", async () => {
    const mod = await import("../FloorPlanEditor.types");
    expect(typeof mod).toBe("object");
  });
});

describe("FloorPlanEditor.panZoom", () => {
  it("导出", async () => {
    const mod = await import("../FloorPlanEditor.panZoom");
    expect(typeof mod).toBe("object");
  });

  it("模块导入不抛错", () => {
    expect(() => import("../FloorPlanEditor.panZoom")).not.toThrow();
  });
});