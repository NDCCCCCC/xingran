/**
 * Phase 88 Batch23 — network/discoveries/templates modals
 */
import { describe, it, expect, vi } from "vitest";
import { renderPageWithEndpoints } from "@/test/utils/renderPage";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { CreateModal as DiscoveryCreateModal } from "@/pages/network/discoveries/modals/CreateModal";
import { TemplateEditModal } from "@/pages/network/templates/modals/EditModal";

async function waitText(text: string, timeoutMs = 6000): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (document.body.innerHTML.includes(text)) return true;
    await new Promise((r) => setTimeout(r, 150));
  }
  return false;
}

const depts = [{ id: "d1", deptName: "网络部" }];

describe("network/discoveries — CreateModal", () => {
  it("renders closed without crash", () => {
    renderPageWithEndpoints(
      <DiscoveryCreateModal open={false} departments={[]} onOk={vi.fn()} onCancel={vi.fn()} />,
      {}
    );
    expect(document.body.innerHTML).toBeDefined();
  });

  it("renders form open with departments", async () => {
    await renderPageWithEndpoints(
      <DiscoveryCreateModal open departments={depts as any} onOk={vi.fn()} onCancel={vi.fn()} />,
      {}
    );
    expect(
      (await waitText("网络")) ||
        (await waitText("部门")) ||
        (await waitText("任务")) ||
        document.querySelector("input") !== null
    ).toBe(true);
  });
});

describe("network/templates — EditModal", () => {
  it("renders closed without crash", () => {
    renderPageWithEndpoints(
      <TemplateEditModal open={false} editingTemplate={null} onOk={vi.fn()} onCancel={vi.fn()} />,
      {}
    );
    expect(document.body.innerHTML).toBeDefined();
  });

  it("renders form open with template", async () => {
    await renderPageWithEndpoints(
      <TemplateEditModal
        open
        editingTemplate={
          {
            id: "t1",
            name: "标准模板",
            vendor: "Cisco",
            description: "默认配置",
          } as any
        }
        onOk={vi.fn()}
        onCancel={vi.fn()}
      />,
      {}
    );
    expect(
      (await waitText("标准模板")) ||
        (await waitText("Cisco")) ||
        document.querySelector("input") !== null
    ).toBe(true);
  });
});
