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
import { useMenuStore } from "@/store/menuStore";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
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

/**
 * Phase 86 W2：operations 8 页接入三态门控后，「权限未就绪」
 * （menuStore 默认 `permissionsLoaded:false`）态下写入口按设计**渲染但 disabled**。
 * 本测试的判定对象是「点新增 → Modal 打开」这条交互分支，因此这里显式构造
 * **ready + 授予本波 ops 页写码**（对无码豁免页零影响 —— 它们的入口不做判定）。
 * ⚠️ 这是按三态语义更新测试前置条件，**未删除任何断言**。
 *
 * Phase 86 W3 追加：`network/devices` 页的「手动新增 / 快速创建」在 W3 接入
 * `networkDevice.add`（走**降级解析** `resolveDeviceActionPerm`：权限集含任一
 * `network:device:*` 动作级码 → 用动作码；否则退回页面级 `network:device`）。
 * 该页的 DevicesPage 用例也走本 helper，故这里必须同批授予 networkDevice 写码，
 * 否则降级解析拿不到动作码 → 按钮不渲染 → helper 找不到可点按钮（与 W2 同一适配口径）。
 */
const OPS_WRITE_PERMISSIONS = [
  PERMISSION_CODES.networkDevice.list,
  PERMISSION_CODES.networkDevice.add,
  PERMISSION_CODES.networkDevice.edit,
  PERMISSION_CODES.networkDevice.delete,
  PERMISSION_CODES.networkDevice.export,
  PERMISSION_CODES.opsBuilding.add,
  PERMISSION_CODES.opsBuilding.edit,
  PERMISSION_CODES.opsBuilding.delete,
  PERMISSION_CODES.opsBuilding.import,
  PERMISSION_CODES.opsBuilding.export,
  PERMISSION_CODES.opsFloor.add,
  PERMISSION_CODES.opsFloor.edit,
  PERMISSION_CODES.opsFloor.delete,
  PERMISSION_CODES.opsFloor.import,
  PERMISSION_CODES.opsFloor.export,
  PERMISSION_CODES.opsWorkstation.add,
  PERMISSION_CODES.opsWorkstation.edit,
  PERMISSION_CODES.opsWorkstation.delete,
  PERMISSION_CODES.opsWorkstation.import,
  PERMISSION_CODES.opsWorkstation.export,
  PERMISSION_CODES.opsInfoPoint.add,
  PERMISSION_CODES.opsInfoPoint.edit,
  PERMISSION_CODES.opsInfoPoint.delete,
  PERMISSION_CODES.opsInfoPoint.import,
  PERMISSION_CODES.opsInfoPoint.export,
  PERMISSION_CODES.opsServerRoom.add,
  PERMISSION_CODES.opsServerRoom.edit,
  PERMISSION_CODES.opsServerRoom.delete,
  PERMISSION_CODES.opsServerRoom.import,
  PERMISSION_CODES.opsServerRoom.export,
  PERMISSION_CODES.opsRoomDevice.add,
  PERMISSION_CODES.opsRoomDevice.edit,
  PERMISSION_CODES.opsRoomDevice.delete,
  PERMISSION_CODES.opsRoomDevice.import,
  PERMISSION_CODES.opsRoomDevice.export,
  PERMISSION_CODES.opsDedicatedLine.add,
  PERMISSION_CODES.opsDedicatedLine.edit,
  PERMISSION_CODES.opsDedicatedLine.delete,
  PERMISSION_CODES.opsDedicatedLine.import,
  PERMISSION_CODES.opsDedicatedLine.export,
  PERMISSION_CODES.opsAsset.add,
  PERMISSION_CODES.opsAsset.edit,
  PERMISSION_CODES.opsAsset.delete,
  PERMISSION_CODES.opsAsset.import,
  PERMISSION_CODES.opsAsset.export,
  PERMISSION_CODES.opsLocationAlias.list,
  PERMISSION_CODES.opsLocationAlias.add,
  PERMISSION_CODES.opsLocationAlias.delete,
  PERMISSION_CODES.opsFloorPlan.edit,
  PERMISSION_CODES.opsWall.edit,
  PERMISSION_CODES.opsDoor.edit,
];

function grantOpsWritePermissions(): void {
  useMenuStore.setState({
    permissions: OPS_WRITE_PERMISSIONS,
    permissionsLoaded: true,
    permissionsError: null,
  });
}

async function openModalAndAssert(
  page: React.ReactElement,
  endpoints: Record<string, unknown> = {}
): Promise<void> {
  // Phase 86 W2：显式 ready 态（否则写入口 disabled，helper 找不到可点按钮）
  grantOpsWritePermissions();
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
