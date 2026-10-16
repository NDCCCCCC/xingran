/**
 * Phase 88 Batch53 — 页面新增 Modal 交互(基建修复后)
 *
 * renderPage 形状基建(departments/tree 等数组端点)修复后页面主 UI
 * 完整渲染,新增按钮可点击 → Modal 表单分支覆盖(每页 +50~140 stmts)。
 */
import { describe, it, expect, vi } from "vitest";
import { fireEvent } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import NoticePage from "../system/notice";
import InfoPointsPage from "../operations/info-points";
import AdUsersPage from "../ad-domain/users";
import FloorsPage from "../operations/floors";
import DevicesPage from "../network/devices";
import BuildingsPage from "../operations/buildings";
import AdGroupsPage from "../ad-domain/groups";
import VirtualMachineList from "../vdi/VirtualMachineList";
import ApiKeysPage from "../system/apikeys";
import BackupsPage from "../network/backups";

async function openModalAndAssert(
  page: React.ReactElement,
  endpoints: Record<string, unknown> = {}
): Promise<void> {
  const { rendered } = renderPageWithEndpoints(page, { endpoints });
  await vi.waitFor(
    () => {
      expect(rendered.container.querySelector(".ant-table, .ant-card, .ant-spin")).not.toBeNull();
    },
    { timeout: 10000 }
  );
  // 等 loading 结束让按钮渲染
  await new Promise((r) => setTimeout(r, 500));
  const btn = Array.from(document.querySelectorAll("button")).find((b) => {
    const t = (b.textContent || "").replace(/\s+/g, "");
    return /新增|新建|添加|手动新增/.test(t) && !(b as HTMLButtonElement).disabled;
  });
  expect(btn).toBeDefined();
  fireEvent.click(btn as HTMLElement);
  await vi.waitFor(
    () => {
      expect(document.querySelector(".ant-modal, .ant-drawer")).not.toBeNull();
    },
    { timeout: 8000 }
  );
}

describe("batch53 新增 Modal 交互", () => {
  it("NoticePage 新增公告 Modal", async () => {
    await openModalAndAssert(<NoticePage />, {
      "/system/notices/list": { data: { list: [], total: 0 } },
    });
  }, 20000);

  it("InfoPointsPage 新增 Modal", async () => {
    await openModalAndAssert(<InfoPointsPage />, {
      "/ops/infoPoint/list": { data: { list: [], total: 0 } },
    });
  }, 20000);

  it("AdUsersPage 基础渲染(无新增按钮)", async () => {
    const { rendered } = renderPageWithEndpoints(<AdUsersPage />, {
      endpoints: { "/ad-domain/users/list": { data: { list: [], total: 0 } } },
    });
    await vi.waitFor(() => {
      expect(rendered.container.querySelector(".ant-table, .ant-card")).not.toBeNull();
    });
  }, 20000);

  it("FloorsPage 新增 Modal", async () => {
    await openModalAndAssert(<FloorsPage />, {
      "/ops/floor/list": { data: { list: [], total: 0 } },
    });
  }, 20000);

  it("DevicesPage 手动新增 Modal", async () => {
    await openModalAndAssert(<DevicesPage />, {
      "/network/devices/list": { data: { list: [], total: 0 } },
    });
  }, 20000);

  it("BuildingsPage 新增 Modal", async () => {
    await openModalAndAssert(<BuildingsPage />, {
      "/ops/building/list": { data: { list: [], total: 0 } },
    });
  }, 20000);

  it("AdGroupsPage 基础渲染(无新增按钮)", async () => {
    const { rendered } = renderPageWithEndpoints(<AdGroupsPage />, {
      endpoints: { "/ad-domain/groups/list": { data: { list: [], total: 0 } } },
    });
    await vi.waitFor(() => {
      expect(rendered.container.querySelector(".ant-table, .ant-card")).not.toBeNull();
    });
  }, 20000);

  it("VirtualMachineList 基础渲染(无新增按钮)", async () => {
    const { rendered } = renderPageWithEndpoints(<VirtualMachineList />, {
      endpoints: { "/vdi/vms/list": { data: { list: [], total: 0 } } },
    });
    await vi.waitFor(() => {
      expect(rendered.container.querySelector(".ant-table, .ant-card")).not.toBeNull();
    });
  }, 20000);

  it("ApiKeysPage 基础渲染(无新增按钮)", async () => {
    const { rendered } = renderPageWithEndpoints(<ApiKeysPage />, {
      endpoints: { "/system/apikeys/list": { data: { list: [], total: 0 } } },
    });
    await vi.waitFor(() => {
      expect(rendered.container.querySelector(".ant-table, .ant-card")).not.toBeNull();
    });
  }, 20000);

  it("BackupsPage 渲染(无新增也可)", async () => {
    const { rendered } = renderPageWithEndpoints(<BackupsPage />, {
      endpoints: { "/network/backups/list": { data: { list: [], total: 0 } } },
    });
    await vi.waitFor(() => {
      expect(rendered.container.querySelector(".ant-table, .ant-card")).not.toBeNull();
    });
  }, 20000);
});
