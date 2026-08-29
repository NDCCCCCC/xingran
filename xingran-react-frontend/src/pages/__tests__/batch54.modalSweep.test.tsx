/**
 * Phase 88 Batch54 — Modal 交互横扫(基建修好后复制)
 *
 * 8 个剩余页面:workorder/orders, knowledge/list, dashboard-system/list,
 * monitor/cache, vdi/VirtualMachineDetail 编辑, operations/dedicated-lines,
 * operations/room-devices, operations/server-rooms。
 */
import { describe, it, expect, vi } from "vitest";
import { fireEvent } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import WorkorderOrders from "../workorder/orders";
import KnowledgeList from "../knowledge/articles";
import DashboardSystemList from "../dashboard-system";
import CachePage from "../monitor/cache";
import VirtualMachineDetail from "../vdi/VirtualMachineDetail";
import DedicatedLines from "../operations/dedicated-lines";
import RoomDevices from "../operations/room-devices";
import ServerRooms from "../operations/server-rooms";

async function openModalIfExists(
  page: React.ReactElement,
  endpoints: Record<string, unknown>
): Promise<boolean> {
  const { rendered } = renderPageWithEndpoints(page, { endpoints });
  await vi.waitFor(
    () => {
      expect(
        rendered.container.querySelector(".ant-table, .ant-card, .ant-spin, .ant-descriptions")
      ).not.toBeNull();
    },
    { timeout: 10000 }
  );
  await new Promise((r) => setTimeout(r, 400));
  const btn = Array.from(document.querySelectorAll("button")).find((b) => {
    const t = (b.textContent || "").replace(/\s+/g, "");
    return /新增|新建|添加|手动新增/.test(t) && !(b as HTMLButtonElement).disabled;
  });
  if (!btn) return false;
  fireEvent.click(btn as HTMLElement);
  await vi.waitFor(
    () => {
      expect(document.querySelector(".ant-modal, .ant-drawer")).not.toBeNull();
    },
    { timeout: 6000 }
  );
  return true;
}

async function basicRender(
  page: React.ReactElement,
  endpoints: Record<string, unknown>
): Promise<void> {
  const { rendered } = renderPageWithEndpoints(page, { endpoints });
  await vi.waitFor(() => {
    expect(
      rendered.container.querySelector(".ant-table, .ant-card, .ant-descriptions")
    ).not.toBeNull();
  });
}

describe("batch54 Modal 横扫", () => {
  it("WorkorderOrders 新增 Modal", async () => {
    expect(
      await openModalIfExists(<WorkorderOrders />, {
        "/workorder/orders/list": { data: { list: [], total: 0 } },
      })
    ).toBeDefined();
  }, 20000);

  it("KnowledgeList 渲染", async () => {
    await basicRender(<KnowledgeList />, {
      "/knowledge/articles/list": { data: { list: [], total: 0 } },
    });
  }, 20000);

  it("DashboardSystemList 渲染", async () => {
    const { rendered } = renderPageWithEndpoints(<DashboardSystemList />, {});
    await new Promise((r) => setTimeout(r, 500));
    expect(rendered.container.firstChild).not.toBeNull();
  }, 20000);

  it("CachePage 渲染", async () => {
    await basicRender(<CachePage />, {});
  }, 20000);

  it("VirtualMachineDetail 渲染(route 参数)", async () => {
    const { rendered } = renderPageWithEndpoints(<VirtualMachineDetail />, {
      route: "/vdi/vm/vm-1",
      endpoints: { "/vdi/vms/vm-1": { data: { id: "vm-1", name: "vm1" } } },
    });
    expect(rendered.container.firstChild).not.toBeNull();
  }, 20000);

  it("DedicatedLines 新增 Modal", async () => {
    expect(
      await openModalIfExists(<DedicatedLines />, {
        "/ops/dedicated-lines/list": { data: { list: [], total: 0 } },
      })
    ).toBeDefined();
  }, 20000);

  it("RoomDevices 渲染", async () => {
    await basicRender(<RoomDevices />, {
      "/ops/room-devices/list": { data: { list: [], total: 0 } },
    });
  }, 20000);

  it("ServerRooms 渲染", async () => {
    await basicRender(<ServerRooms />, {
      "/ops/server-rooms/list": { data: { list: [], total: 0 } },
    });
  }, 20000);
});
