/**
 * Phase 85 — info-points 页面模块导入断言
 */
import { describe, it, expect } from "vitest";
import InfoPointManagement from "../index";

describe("info-points page module", () => {
  it("exports default page component", () => {
    expect(InfoPointManagement).toBeDefined();
    expect(typeof InfoPointManagement).toBe("function");
  });
});
