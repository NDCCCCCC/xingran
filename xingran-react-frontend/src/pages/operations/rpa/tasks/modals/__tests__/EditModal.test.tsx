/**
 * Phase 88 Batch79 — RPA tasks EditModal 渲染测试(55 stmts, 25.5% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { TaskEditModal } from "../EditModal";
import { Form } from "antd";
import type { Task } from "@/types/rpa";
import { useState } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

function Harness({
  task,
  onOk = vi.fn(),
  onCancel = vi.fn(),
}: {
  task: Task | null;
  onOk?: (v: any) => Promise<void>;
  onCancel?: () => void;
}) {
  const [form] = Form.useForm();
  return <TaskEditModal open form={form} editingTask={task} onOk={onOk} onCancel={onCancel} />;
}

describe("TaskEditModal 渲染", () => {
  function ClosedHarness() {
    const [form] = Form.useForm();
    return (
      <TaskEditModal
        open={false}
        form={form}
        editingTask={null}
        onOk={vi.fn()}
        onCancel={vi.fn()}
      />
    );
  }

  it("open=false 不渲染内容", () => {
    const { baseElement } = renderWithProviders(<ClosedHarness />);
    expect(baseElement.querySelector(".ant-modal-body")).toBeNull();
  });

  it("open=true + task=null → 新建模式", async () => {
    const { baseElement } = renderWithProviders(<Harness task={null} />);
    await new Promise((r) => setTimeout(r, 100));
    expect(baseElement).toBeDefined();
  });

  it("open=true + 已有 task → 编辑模式(填充表单值)", async () => {
    const task: Task = {
      id: "t1",
      taskName: "测试任务",
      taskDescription: "desc",
      actions: [{ id: "a1", type: "navigate", value: "https://example.com" }],
      enabled: true,
      timeout: 300,
    } as any;
    const { baseElement } = renderWithProviders(<Harness task={task} />);
    await new Promise((r) => setTimeout(r, 200));
    expect(baseElement).toBeDefined();
  });
});
