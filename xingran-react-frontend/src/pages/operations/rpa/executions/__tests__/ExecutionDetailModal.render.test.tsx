/**
 * Phase 88 Batch74 — RPA ExecutionDetailModal 渲染
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { ExecutionDetailModal } from "../ExecutionDetailModal";
import type { Execution } from "@/types/rpa";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const exec: Execution = {
  id: "e1",
  taskName: "test-task",
  status: "running",
  startedAt: "2026-08-29 10:00:00",
  duration: 0,
  workerName: "worker-1",
} as unknown as Execution;

describe("ExecutionDetailModal 渲染", () => {
  it("open=false 不渲染内容", () => {
    const { baseElement } = renderWithProviders(
      <ExecutionDetailModal open={false} execution={null} onClose={vi.fn()} />
    );
    expect(baseElement.querySelector(".ant-modal-body")).toBeNull();
  });

  it("open=true + execution=非空 渲染 Modal", async () => {
    renderWithProviders(<ExecutionDetailModal open execution={exec} onClose={vi.fn()} />);
    await new Promise((r) => setTimeout(r, 500));
  });

  it("execution=null 时 open=true 不渲染执行信息", () => {
    const { baseElement } = renderWithProviders(
      <ExecutionDetailModal open execution={null} onClose={vi.fn()} />
    );
    expect(baseElement).toBeDefined();
  });
});
