/**
 * Phase 86 W3（RBAC-02）—— 配置备份页操作级权限门控测试。
 *
 * 断言对象是**非标准动作**「还原」「版本对比」（本页无删除入口）：
 *   ① 只持 `networkBackup.list` → 还原 / 版本对比 / 创建备份 / 导出**不在 DOM**；
 *      只读入口「版本列表」保持无门控（仍在 DOM）；
 *   ② 未就绪（permissionsLoaded=false / permissionsError 非空）→ 入口**在 DOM 且 disabled**
 *      且带提示（title 含「权限」）；
 *   ③ 持 `.restore` → 「还原」在、「版本对比」不在；持 `.diff` → 「版本对比」在；
 *   ④ ready + 空权限集 → 不渲染（与「未就绪」可区分）。
 *
 * ⚠️ **MEDIUM-4 回归防线（本文件最重要的一条）**：
 *    持 `networkBackup.restore` 但**无** `networkBackup.delete` 时，「还原」必须**仍可用**。
 *    本页的「还原确认」复用了 `confirmDelete` 这个二次确认 helper（形参名 `onDelete`
 *    与业务语义无关，`title="确认恢复配置"` / `actionLabel="恢复"` / `successText="已恢复配置"`
 *    才是语义证据）。若把还原误挂 `networkBackup.delete`，会出现双向错误：
 *      ① 持 restore 无 delete 者**被误拒**（有权还原却看不到按钮）—— 本用例锁定这一侧；
 *      ② 持 delete 无 restore 者**可越权还原**（把配置恢复到任意历史版本）—— 由
 *         「③-2 持 delete 无 restore → 还原不在 DOM」锁定另一侧。
 *
 * ⚠️ 行操作有 3 个（还原 / 版本列表 / 版本对比）→ ActionButtons threshold=3 会收进
 *    「操作」下拉；断言需先点开下拉。只读入口「版本列表」无 permission → 恒可见。
 * ⚠️ 断言只用 vitest 原生 matcher + 原生 disabled 属性，不用 jest-dom（理由见 credentials 同名文件）。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { screen, waitFor, cleanup, fireEvent } from "@testing-library/react";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { mockApiBatch, setGenericFallback } from "@/test/utils/createApiMock";
import { useMenuStore } from "@/store/menuStore";
import { PERMISSION_CODES } from "@/constants/permissionCodes";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/api/networkApi", () => ({
  batchExport: vi.fn(),
  getNetworkConfig: vi.fn(() => Promise.resolve({ data: {} })),
  getNetworkConfigDiff: vi.fn(() => Promise.resolve({ data: { items: [] } })),
}));

import BackupPage from "../index";

/**
 * 备份列表行 → 经 `groupBackupsByDevice` 分组后成为设备分组行。
 * 需要至少 2 份备份，「版本对比」才有意义（否则页面会 message.info 拦截）。
 */
const BACKUP_ROWS = [
  {
    id: "b2",
    deviceId: "d1",
    deviceName: "switch-01",
    ipAddress: "10.0.0.1",
    version: "v2",
    backupType: "auto",
    backupSize: 2048,
    changeReason: "例行备份",
    createdBy: "admin",
    createdAt: "2026-01-02 12:00:00",
  },
  {
    id: "b1",
    deviceId: "d1",
    deviceName: "switch-01",
    ipAddress: "10.0.0.1",
    version: "v1",
    backupType: "manual",
    backupSize: 1024,
    changeReason: "首次备份",
    createdBy: "admin",
    createdAt: "2026-01-01 12:00:00",
  },
];

/** 设置权限集与显式就绪位（构造 ready / loading / error 三态） */
function setPermissionState(
  permissions: string[],
  state: { loaded?: boolean; error?: Error } = {}
): void {
  useMenuStore.setState({
    permissions,
    permissionsLoaded: state.loaded ?? true,
    permissionsError: state.error ?? null,
  });
}

function renderPage() {
  return renderWithProviders(<BackupPage />);
}

/** 点开行操作「操作」下拉，返回其菜单容器（antd Dropdown 挂 body） */
async function openRowActionsMenu(): Promise<Element> {
  fireEvent.click(screen.getByRole("button", { name: "操作" }));
  await waitFor(() => {
    expect(document.querySelector(".ant-dropdown")).toBeTruthy();
  });
  const menu = document.querySelector(".ant-dropdown");
  if (!menu) throw new Error("行操作下拉未渲染");
  return menu;
}

describe("配置备份页操作级权限门控（Phase 86 W3）", () => {
  beforeEach(() => {
    mockApiBatch([
      {
        endpoint: "/network/backups/list",
        response: { data: { list: BACKUP_ROWS, total: 2 } },
      },
      {
        endpoint: "/network/backups/statistics",
        response: { data: { total: 2, auto: 1, manual: 1, devices: 1 } },
      },
      { endpoint: "/network/devices/list", response: { data: { list: [], total: 0 } } },
    ]);
    setGenericFallback({ data: {} });
  });

  // confirmDelete 走命令式 Modal.confirm（独立挂 body），RTL cleanup 不覆盖——手动清残留
  afterEach(() => {
    cleanup();
    document.body
      .querySelectorAll(".ant-modal-root, .ant-modal-mask, .ant-dropdown")
      .forEach((node) => node.remove());
  });

  // ==================== ① ready + 只持读码 ====================

  it("① 只持 networkBackup.list → 创建备份/导出不在 DOM；行「还原」「版本对比」不在下拉内，只读「版本列表」保留", async () => {
    setPermissionState([PERMISSION_CODES.networkBackup.list]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("switch-01")).toBeTruthy();
    });

    // 工具栏写入口
    expect(screen.queryByRole("button", { name: /创建备份/ })).toBeNull();
    // 批量导出：门控点 = <NetworkExport> 组件元素本身
    expect(screen.queryByRole("button", { name: /导出/ })).toBeNull();

    // 行操作：还原 / 版本对比被门控掉；只剩无门控的只读「版本列表」→
    // visible.length = 1 < threshold(3) → 直接平铺渲染，不再收进「操作」下拉。
    expect(screen.queryByRole("button", { name: "操作" })).toBeNull();
    expect(screen.getByRole("button", { name: /版本列表/ })).toBeTruthy();
    expect(screen.queryByRole("button", { name: /还原/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /版本对比/ })).toBeNull();
  });

  // ==================== ④ ready + 空权限集 ====================

  it("④ ready 且空权限集 → 写入口不渲染，只读入口仍在（与「未就绪」可区分）", async () => {
    setPermissionState([], { loaded: true });
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("switch-01")).toBeTruthy();
    });

    expect(screen.queryByRole("button", { name: /创建备份/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /导出/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /还原/ })).toBeNull();
    expect(screen.getByRole("button", { name: /版本列表/ })).toBeTruthy();
  });

  // ==================== ② 未就绪（loading / error） ====================

  it("② 未就绪（loading）→ 创建备份/导出在 DOM 且 disabled + 提示；行操作仍可见但禁用", async () => {
    setPermissionState([], { loaded: false });
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("switch-01")).toBeTruthy();
    });

    const createBtn = screen.getByRole("button", { name: /创建备份/ }) as HTMLButtonElement;
    expect(createBtn.disabled).toBe(true);
    expect(createBtn.getAttribute("title") ?? "").toContain("权限");

    const exportEntry = screen.getByRole("button", { name: /导出/ });
    expect(exportEntry.closest('[aria-disabled="true"]')).toBeTruthy();
    expect(exportEntry.closest('[aria-disabled="true"]')?.getAttribute("title") ?? "").toContain(
      "权限"
    );

    // 行操作未就绪 → 入口仍在 DOM（下拉存在），菜单项 disabled + 带提示
    const menu = await openRowActionsMenu();
    const text = menu.textContent ?? "";
    expect(text).toContain("还原");
    expect(text).toContain("版本对比");
    expect(menu.querySelectorAll(".ant-dropdown-menu-item-disabled").length).toBeGreaterThan(0);
  });

  it("②-2 未就绪（error）→ 入口在 DOM 且 disabled + 提示", async () => {
    setPermissionState([], { loaded: false, error: new Error("权限接口失败") });
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("switch-01")).toBeTruthy();
    });

    const createBtn = screen.getByRole("button", { name: /创建备份/ }) as HTMLButtonElement;
    expect(createBtn.disabled).toBe(true);
    const exportEntry = screen.getByRole("button", { name: /导出/ });
    expect(exportEntry.closest('[aria-disabled="true"]')).toBeTruthy();
  });

  // ==================== ③ ready + 单码（还原 / 对比分别归码） ====================

  it("③ 持 restore（无 diff）→ 「还原」在、「版本对比」不在；「版本列表」仍可见", async () => {
    setPermissionState([
      PERMISSION_CODES.networkBackup.list,
      PERMISSION_CODES.networkBackup.restore,
    ]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("switch-01")).toBeTruthy();
    });

    // 还原 + 版本列表 = 2 个可见动作 < threshold(3) → 平铺渲染
    expect(screen.queryByRole("button", { name: "操作" })).toBeNull();
    expect(screen.getByRole("button", { name: /还原/ })).toBeTruthy();
    expect(screen.getByRole("button", { name: /版本列表/ })).toBeTruthy();
    expect(screen.queryByRole("button", { name: /版本对比/ })).toBeNull();
  });

  it("③-2 持 diff（无 restore）→ 「版本对比」在、「还原」不在", async () => {
    setPermissionState([PERMISSION_CODES.networkBackup.list, PERMISSION_CODES.networkBackup.diff]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("switch-01")).toBeTruthy();
    });

    expect(screen.getByRole("button", { name: /版本对比/ })).toBeTruthy();
    expect(screen.getByRole("button", { name: /版本列表/ })).toBeTruthy();
    expect(screen.queryByRole("button", { name: /还原/ })).toBeNull();
  });

  // ==================== MEDIUM-4 回归防线 ====================

  it("③-3 【越权回归】持 restore 但无 delete → 「还原」仍可用（还原不得被误挂 delete）", async () => {
    // ⚠️ 这是本文件的回归防线：本页无删除入口，`networkBackup.delete` 不参与任何门控。
    // 若有人把「还原确认」误分类成删除（因为 `confirmDelete` 形参名诱导），
    // 该用例立刻变红：持 restore 无 delete 的用户会被误拒。
    setPermissionState([
      PERMISSION_CODES.networkBackup.list,
      PERMISSION_CODES.networkBackup.restore,
      // 故意**不**给 networkBackup.delete
    ]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("switch-01")).toBeTruthy();
    });

    const restoreBtn = screen.getByRole("button", { name: /还原/ }) as HTMLButtonElement;
    expect(restoreBtn).toBeTruthy();
    expect(restoreBtn.disabled).toBe(false);
    // 反向证据：本页无删除入口 → DOM 中不存在「删除」行按钮
    expect(screen.queryByRole("button", { name: /^删除/ })).toBeNull();
  });

  it("③-4 【越权回归反向】持 delete 但无 restore → 「还原」不在 DOM（delete 不能放行还原）", async () => {
    // 另一侧的双向错误：若还原误挂 delete，持 delete 无 restore 者可越权还原。
    setPermissionState([
      PERMISSION_CODES.networkBackup.list,
      PERMISSION_CODES.networkBackup.delete,
    ]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("switch-01")).toBeTruthy();
    });

    expect(screen.queryByRole("button", { name: /还原/ })).toBeNull();
    expect(screen.getByRole("button", { name: /版本列表/ })).toBeTruthy();
  });

  // ==================== ③-5 全码 ====================

  it("③-5 持 add/restore/diff/export → 创建备份/导出/还原/版本对比全部存在且可用", async () => {
    setPermissionState([
      PERMISSION_CODES.networkBackup.list,
      PERMISSION_CODES.networkBackup.add,
      PERMISSION_CODES.networkBackup.restore,
      PERMISSION_CODES.networkBackup.diff,
      PERMISSION_CODES.networkBackup.export,
    ]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("switch-01")).toBeTruthy();
    });

    const createBtn = screen.getByRole("button", { name: /创建备份/ }) as HTMLButtonElement;
    expect(createBtn.disabled).toBe(false);
    const exportBtn = screen.getByRole("button", { name: /导出/ }) as HTMLButtonElement;
    expect(exportBtn.disabled).toBe(false);

    // 3 个行操作全可见 → 收进「操作」下拉
    const menu = await openRowActionsMenu();
    const text = menu.textContent ?? "";
    expect(text).toContain("还原");
    expect(text).toContain("版本列表");
    expect(text).toContain("版本对比");
    expect(menu.querySelectorAll(".ant-dropdown-menu-item-disabled").length).toBe(0);
  });
}, 30000);
