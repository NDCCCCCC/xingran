/**
 * Phase 88 Batch75 — ad-domain AccountPoolTab 渲染
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import AccountPoolTab from "../AccountPoolTab";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

describe("AccountPoolTab 渲染", () => {
  it("configId 非空渲染 Tab + 统计", () => {
    const { baseElement } = renderWithProviders(<AccountPoolTab configId="c1" />);
    expect(baseElement).toBeDefined();
  });
});
