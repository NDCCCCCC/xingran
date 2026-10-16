/**
 * Phase 86 — apikeys 页面模块导入断言
 */
import { describe, it, expect } from "vitest";
import ApiKeysPage from "../index";

describe("apikeys page module", () => {
  it("exports default page component", () => {
    expect(ApiKeysPage).toBeDefined();
    expect(typeof ApiKeysPage).toBe("function");
  });
});
