/**
 * Phase 86 — config 页面模块导入断言
 */
import { describe, it, expect } from "vitest";
import ConfigPage from "../index";

describe("config page module", () => {
  it("exports default page component", () => {
    expect(ConfigPage).toBeDefined();
  });
});
