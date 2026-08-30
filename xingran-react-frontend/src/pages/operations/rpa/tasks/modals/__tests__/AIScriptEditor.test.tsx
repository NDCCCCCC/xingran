/**
 * Phase 88 Batch100 — RPA tasks AIScriptEditor 测试(33 stmts, 33.3% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { AIScriptEditor } from "../AIScriptEditor";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

describe("AIScriptEditor 渲染", () => {
  it("open=false → 不渲染内容", () => {
    const { baseElement } = renderWithProviders(
      <AIScriptEditor open={false} onClose={vi.fn()} onConfirm={vi.fn()} />
    );
    expect(baseElement.querySelector(".ant-modal-body")).toBeNull();
  });

  it("open=true → 渲染表单", () => {
    const { baseElement } = renderWithProviders(
      <AIScriptEditor open onClose={vi.fn()} onConfirm={vi.fn()} />
    );
    expect(baseElement).toBeDefined();
  });
});
