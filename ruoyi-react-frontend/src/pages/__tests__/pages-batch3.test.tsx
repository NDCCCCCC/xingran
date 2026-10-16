/**
 * Phase 88 — 第三批页面渲染: 子页面深挖
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import MenuManagement from "../system/menu";
import DepartmentManagement from "../system/dept";
import RoleManagement from "../system/role";
import DeviceDiscoveryPage from "../network/discoveries";
import ConfigExecutionPage from "../network/executions";
import ConfigBackupPage from "../network/backups";
import DutySchedulePage from "../duty/schedules";
import DutyPoolPage from "../duty/pools";
import AssetReconciliation from "../asset/reconciliation";

async function expectRenders(ui: React.ReactElement, endpoints: Record<string, unknown> = {}) {
  const { rendered } = renderPageWithEndpoints(ui, { endpoints });
  await vi.waitFor(
    () => {
      expect(rendered.container.firstChild).not.toBeNull();
    },
    { timeout: 8000 }
  );
  return rendered;
}

describe("第三批子页面渲染(真实 hooks)", () => {
  it("system/menu 页", async () => {
    await expectRenders(<MenuManagement />, {
      "/system/menus/list": { data: { list: [], total: 0 } },
    });
  });
  it("system/dept 页", async () => {
    await expectRenders(<DepartmentManagement />, { "/system/departments/tree": { data: [] } });
  });
  it("system/role 页", async () => {
    await expectRenders(<RoleManagement />, {
      "/system/roles/list": { data: { list: [], total: 0 } },
    });
  });
  it("network/discoveries 页", async () => {
    await expectRenders(<DeviceDiscoveryPage />, {});
  });
  it("network/executions 页", async () => {
    await expectRenders(<ConfigExecutionPage />, {});
  });
  it("network/backups 页", async () => {
    await expectRenders(<ConfigBackupPage />, {});
  });
  it("duty/schedules 页", async () => {
    await expectRenders(<DutySchedulePage />, {});
  });
  it("duty/pools 页", async () => {
    await expectRenders(<DutyPoolPage />, {});
  });
  it("asset/reconciliation 页", async () => {
    await expectRenders(<AssetReconciliation />, {});
  });
});
