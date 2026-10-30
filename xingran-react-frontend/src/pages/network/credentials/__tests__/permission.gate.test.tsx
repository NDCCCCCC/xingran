/**
 * Phase 86 W3（RBAC-02）—— 设备凭据页操作级权限门控测试。
 *
 * 覆盖 PLAN 3.4 的四条路径（三路径 + 「ready 空权限集 vs 未就绪」可区分）：
 *   ① 只持 `networkCredential.list` → 新增 / 编辑 / 删除 / 「设为默认」/ 批量删除 /
 *      「批量导出」（`<NetworkExport>` 组件内的「导出」按钮）均**不在 DOM**；
 *   ② 未就绪（permissionsLoaded=false / permissionsError 非空）→ 入口**在 DOM 且 disabled**
 *      且带提示（title 含「权限」）；
 *   ③ 持 `.add/.edit/.delete/.export` → 入口存在且可用；
 *   ④ ready + **空权限集** → 不渲染（与「未就绪」在数据上可区分）。
 *
 * ⚠️ 禁止用 `permissions.length === 0` 判「未就绪」——空权限集与未加载完在数据上不可区分。
 *    本测试用 menuStore 的显式就绪位（permissionsLoaded / permissionsError）构造三态。
 * ⚠️ 断言只用 vitest 原生 matcher（toBeNull / toBe / toContain）+ 原生 disabled 属性，
 *    不用 jest-dom matcher —— `tsconfig.app.json` 排除了测试文件，jest-dom 类型不在
 *    专项类型检查的 types 里，用它会引入类型错误（本文件要求类型错误数 = 0）。
 * ⚠️ 本页行操作有 3 个（设为默认 / 编辑 / 删除）→ ActionButtons 的 threshold=3 会
 *    把它们**收进「操作」下拉**（`ActionButtons.tsx` 的 `visible.length >= threshold` 分支）。
 *    因此行操作用例必须先点开「操作」再断言菜单项，不能直接 `getByRole("button", {name:"编辑"})`。
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
}));

import CredentialPage from "../index";

const CREDENTIAL_ROW = {
  id: "c1",
  credentialName: "cred-01",
  protocolType: "ssh",
  username: "admin",
  isDefault: false,
  createdAt: "2026-01-01 12:00:00",
};

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
  return renderWithProviders(<CredentialPage />);
}

/** 断言某按钮「在 DOM 且 disabled，且 title 带未就绪提示」 */
function expectDisabledWithHint(name: string | RegExp): void {
  const btn = screen.getByRole("button", { name }) as HTMLButtonElement;
  expect(btn).toBeTruthy();
  expect(btn.disabled).toBe(true);
  expect(btn.getAttribute("title") ?? "").toContain("权限");
}

/** 点开行操作「操作」下拉，返回其菜单容器（antd Dropdown 挂 body） */
async function openRowActionsMenu(): Promise<Element> {
  fireEvent.click(screen.getByRole("button", { name: "操作" }));
  await waitFor(() => {
    const menu = document.querySelector(".ant-dropdown");
    expect(menu).toBeTruthy();
  });
  const menu = document.querySelector(".ant-dropdown");
  if (!menu) throw new Error("行操作下拉未渲染");
  return menu;
}

describe("设备凭据页操作级权限门控（Phase 86 W3）", () => {
  beforeEach(() => {
    mockApiBatch([
      {
        endpoint: "/network/credentials/list",
        response: { data: { list: [CREDENTIAL_ROW], total: 1 } },
      },
      {
        endpoint: "/network/credentials/statistics",
        response: { data: { total: 1, ssh: 1, telnet: 0 } },
      },
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

  it("① 只持 networkCredential.list → 新增/行操作/批量删除/导出均不在 DOM", async () => {
    setPermissionState([PERMISSION_CODES.networkCredential.list]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("cred-01")).toBeTruthy();
    });

    // 工具栏写入口
    expect(screen.queryByRole("button", { name: /新增/ })).toBeNull();
    // 行操作：三个动作全被门控掉 → ActionButtons 的 visible 为空 → 组件返回 null
    //（连「操作」下拉触发器都不渲染，这是最强的「不在 DOM」证据）
    expect(screen.queryByRole("button", { name: "操作" })).toBeNull();
    // 批量删除（选中后才会出现；此处断言其不存在即可）
    expect(screen.queryByRole("button", { name: /批量删除/ })).toBeNull();
    // 批量导出：门控点 = <NetworkExport> 组件元素本身 → 组件不在 DOM，
    // 其内部「导出」按钮自然也不在 DOM。
    expect(screen.queryByRole("button", { name: /导出/ })).toBeNull();
  });

  // ==================== ④ ready + 空权限集 ====================

  it("④ ready 且空权限集 → 入口不渲染（与「未就绪」可区分）", async () => {
    setPermissionState([], { loaded: true });
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("cred-01")).toBeTruthy();
    });

    expect(screen.queryByRole("button", { name: /新增/ })).toBeNull();
    expect(screen.queryByRole("button", { name: "操作" })).toBeNull();
    expect(screen.queryByRole("button", { name: /导出/ })).toBeNull();
  });

  // ==================== ② 未就绪（loading / error） ====================

  it("② 未就绪（loading）→ 新增/导出/行操作在 DOM 且 disabled + 提示", async () => {
    // permissions 为空但 permissionsLoaded=false —— 这正是「未就绪」态
    setPermissionState([], { loaded: false });
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("cred-01")).toBeTruthy();
    });

    expectDisabledWithHint(/新增/);
    // 导出按钮在 <NetworkExport> 组件内部（未就绪 → 组件仍渲染，但被 aria-disabled 包裹）
    const exportEntry = screen.getByRole("button", { name: /导出/ });
    expect(exportEntry.closest('[aria-disabled="true"]')).toBeTruthy();
    expect(exportEntry.closest('[aria-disabled="true"]')?.getAttribute("title") ?? "").toContain(
      "权限"
    );

    // 行操作：未就绪 → 入口仍在 DOM（「操作」下拉存在），但菜单项全部 disabled + 带提示
    const menu = await openRowActionsMenu();
    const text = menu.textContent ?? "";
    expect(text).toContain("设为默认");
    expect(text).toContain("编辑");
    expect(text).toContain("删除");
    expect(menu.querySelectorAll(".ant-dropdown-menu-item-disabled").length).toBeGreaterThan(0);
  });

  it("②-2 未就绪（error）→ 入口在 DOM 且 disabled + 提示", async () => {
    setPermissionState([], { loaded: false, error: new Error("权限接口失败") });
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("cred-01")).toBeTruthy();
    });

    expectDisabledWithHint(/新增/);
    const exportEntry = screen.getByRole("button", { name: /导出/ });
    expect(exportEntry.closest('[aria-disabled="true"]')).toBeTruthy();
    const menu = await openRowActionsMenu();
    expect(menu.querySelectorAll(".ant-dropdown-menu-item-disabled").length).toBeGreaterThan(0);
  });

  // ==================== ③ ready + 全写码 ====================

  it("③ 持 add/edit/delete/export → 入口存在且可用（设为默认归 edit）", async () => {
    setPermissionState([
      PERMISSION_CODES.networkCredential.list,
      PERMISSION_CODES.networkCredential.add,
      PERMISSION_CODES.networkCredential.edit,
      PERMISSION_CODES.networkCredential.delete,
      PERMISSION_CODES.networkCredential.export,
    ]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("cred-01")).toBeTruthy();
    });

    const addBtn = screen.getByRole("button", { name: /新增/ }) as HTMLButtonElement;
    expect(addBtn.disabled).toBe(false);
    const exportBtn = screen.getByRole("button", { name: /导出/ }) as HTMLButtonElement;
    expect(exportBtn.disabled).toBe(false);

    // 行操作：持 edit → 「设为默认」（就近归 edit）与「编辑」可见；
    // 持 delete → 「删除」可见；且菜单项均未禁用。
    const menu = await openRowActionsMenu();
    const text = menu.textContent ?? "";
    expect(text).toContain("设为默认");
    expect(text).toContain("编辑");
    expect(text).toContain("删除");
    expect(menu.querySelectorAll(".ant-dropdown-menu-item-disabled").length).toBe(0);
  });
}, 30000);
