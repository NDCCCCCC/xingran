/**
 * Phase 86 — post 页面模块导入断言
 */
import { describe, it, expect } from "vitest";
import PostPage from "../index";

describe("post page module", () => {
  it("exports default page component", () => {
    expect(PostPage).toBeDefined();
  });
});
