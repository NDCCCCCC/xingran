/**
 * Phase 86 — credentials 页面模块导入断言
 */
import { describe, it, expect } from "vitest";
import CredentialsPage from "../index";

describe("credentials page module", () => {
  it("exports default page component", () => {
    expect(CredentialsPage).toBeDefined();
  });
});
