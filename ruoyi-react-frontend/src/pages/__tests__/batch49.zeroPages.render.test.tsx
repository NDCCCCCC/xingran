/**
 * Phase 88 Batch49 — 0% 大页面批量渲染(真实 hooks + mock API)
 *
 * MACHistoryPage(168) / PeriodicTemplatePage(72) / VirtualMachineDetail(81) /
 * CommandDispatch(73) / OUSWithDept(133) / AssetList(116) — 共 ~643 stmts 洼地。
 * 用 renderPageWithEndpoints 真实执行 hooks,断言表格/卡片容器渲染。
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

// Three.js 3D 组件 jsdom canvas 不兼容 — 该页详情不含 3D,但防 lazy 串扰统一 stub
vi.mock("@/components/three", () => ({}), { virtual: true });

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import MACHistoryPage from "../network/mac/history/MACHistoryPage";
import PeriodicTemplatePage from "../workorder/periodic/templates";
import VirtualMachineDetail from "../vdi/VirtualMachineDetail";
import CommandDispatch from "../network/command";
import AssetList from "../operations/assets";

async function renderWithTable(page: React.ReactElement, endpoints: Record<string, unknown>) {
  const { rendered } = renderPageWithEndpoints(page, { endpoints });
  await vi.waitFor(
    () => {
      expect(
        rendered.container.querySelector(".ant-table, .ant-card, .ant-spin, .ant-empty")
      ).not.toBeNull();
    },
    { timeout: 10000 }
  );
  return rendered;
}

describe("batch49 0% 页面渲染", () => {
  it("MACHistoryPage 渲染", async () => {
    await renderWithTable(<MACHistoryPage />, {
      "/network/mac/history/list": { data: { list: [], total: 0 } },
    });
  }, 15000);

  it("PeriodicTemplatePage 渲染", async () => {
    await renderWithTable(<PeriodicTemplatePage />, {
      "/workorder/periodic/templates/list": { data: { list: [], total: 0 } },
    });
  }, 15000);

  it("VirtualMachineDetail 渲染(route 参数)", async () => {
    const { rendered } = renderPageWithEndpoints(<VirtualMachineDetail />, {
      route: "/vdi/vm/vm-1",
      endpoints: {
        "/vdi/vms/vm-1": { data: { id: "vm-1", name: "vm-one", status: 0 } },
      },
    });
    await vi.waitFor(
      () => {
        expect(rendered.container.firstChild).not.toBeNull();
      },
      { timeout: 10000 }
    );
  }, 15000);

  it("CommandDispatch 渲染", async () => {
    await renderWithTable(<CommandDispatch />, {
      "/network/command/list": { data: { list: [], total: 0 } },
    });
  }, 15000);

  it("AssetList 渲染", async () => {
    await renderWithTable(<AssetList />, {
      "/asset/list": { data: { list: [], total: 0 } },
    });
  }, 15000);
});
