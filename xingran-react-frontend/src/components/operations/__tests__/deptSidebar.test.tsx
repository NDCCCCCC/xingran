/**
 * Phase 84 84-02b — DeptSidebar 静态断言(mock DeptTree)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { DeptSidebar } from "../DeptSidebar";

describe("DeptSidebar", () => {
  it("imports without error", () => {
    expect(DeptSidebar).toBeDefined();
    expect(typeof DeptSidebar).toBe("function");
  });
});
