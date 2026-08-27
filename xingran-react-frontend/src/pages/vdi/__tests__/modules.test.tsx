/**
 * Phase 87 — vdi 3 子页面模块导入断言
 */
import { describe, it, expect } from "vitest";

describe("vdi page modules", () => {
  it("VDIServerConfig imports", async () => {
    const m = await import("../VDIServerConfig");
    expect(m.default).toBeDefined();
  });
  it("VirtualMachineList imports", async () => {
    const m = await import("../VirtualMachineList");
    expect(m.default).toBeDefined();
  });
});
