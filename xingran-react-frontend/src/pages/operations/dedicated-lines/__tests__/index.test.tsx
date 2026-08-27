/**
 * Phase 85 — dedicated-lines 页面模块导入断言
 */
import { describe, it, expect } from "vitest";
import DedicatedLine from "../index";

describe("dedicated-lines page module", () => {
  it("exports default page component", () => {
    expect(DedicatedLine).toBeDefined();
    expect(typeof DedicatedLine).toBe("function");
  });
});
