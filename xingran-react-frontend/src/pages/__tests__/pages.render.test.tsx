/**
 * Phase 88 — duty/ad-domain/vdi/workorder/knowledge 页面批量渲染
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import MyDutyPage from "../duty/my-duty";
import DutyManagement from "../duty/management";
import ADUserPage from "../ad-domain/users";
import ADGroupPage from "../ad-domain/groups";
import VirtualMachineList from "../vdi/VirtualMachineList";
import WorkOrderPage from "../workorder/orders";
import KnowledgeArticlePage from "../knowledge/articles";

const emptyList = { data: { list: [], total: 0 } };

async function expectRenders(ui: React.ReactElement, endpoints: Record<string, unknown> = {}) {
  const { rendered } = renderPageWithEndpoints(ui, { endpoints });
  await vi.waitFor(
    () => {
      expect(
        rendered.container.querySelector(
          ".ant-table, .ant-card, .ant-empty, .ant-calendar, .ant-tabs"
        )
      ).not.toBeNull();
    },
    { timeout: 8000 }
  );
  return rendered;
}

describe("大页面批量渲染(真实 hooks)", () => {
  it("duty/my-duty 页", async () => {
    await expectRenders(<MyDutyPage />, { "/duty/my-duty/stats": { data: {} } });
  });

  it("duty/management 页", async () => {
    await expectRenders(<DutyManagement />, {});
  });

  it("ad-domain/users 页", async () => {
    await expectRenders(<ADUserPage />, {});
  });

  it("ad-domain/groups 页", async () => {
    await expectRenders(<ADGroupPage />, {});
  });

  it("vdi/VirtualMachineList 页", async () => {
    await expectRenders(<VirtualMachineList />, {});
  });

  it("workorder/orders 页", async () => {
    await expectRenders(<WorkOrderPage />, {});
  });

  it("knowledge/articles 页", async () => {
    await expectRends_Knowledge();
  });
});

async function expectRends_Knowledge() {
  const { rendered } = renderPageWithEndpoints(<KnowledgeArticlePage />, {});
  await vi.waitFor(
    () => {
      expect(rendered.container.firstChild).not.toBeNull();
    },
    { timeout: 8000 }
  );
}
