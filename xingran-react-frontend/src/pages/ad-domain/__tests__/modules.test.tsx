/**
 * Phase 87 — ad-domain 6 子页面模块导入断言
 */
import { describe, it, expect } from "vitest";

describe("ad-domain page modules", () => {
  it("computers page imports", async () => {
    const m = await import("../computers");
    expect(m.default).toBeDefined();
  });

  it("groups page imports", async () => {
    const m = await import("../groups");
    expect(m.default).toBeDefined();
  });

  it("ous page imports", async () => {
    const m = await import("../ous");
    expect(m.default).toBeDefined();
  });

  it("users page imports", async () => {
    const m = await import("../users");
    expect(m.default).toBeDefined();
  });

  it("logs page imports", async () => {
    const m = await import("../logs");
    expect(m.default).toBeDefined();
  });

  it("configs page imports", async () => {
    const m = await import("../configs");
    expect(m.default).toBeDefined();
  });
});
