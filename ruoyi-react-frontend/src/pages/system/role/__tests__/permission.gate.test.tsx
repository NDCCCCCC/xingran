/**
 * Phase 86 W1（RBAC-02）—— 角色管理页操作级权限门控测试。
 *
 * 覆盖三条路径（+ ready 空权限集第四路径）：
 *   A5 无权限不在 DOM：只持 system:role:list → 新增 / 行编辑 / 行停用 / 行删除 / 批量删除均不存在；
 *   A6 未就绪 disabled：permissionsLoaded=false → 入口仍在 DOM 且 disabled + 带提示（title）；
 *   A7 有权限可点：持 add/edit/remove → 对应入口存在且未禁用。
 *
 * ⚠️ 禁止用 `permissions.length === 0` 判「未就绪」——空权限集与未加载完在数据上
 *    不可区分。本测试用 menuStore 的显式就绪位（permissionsLoaded / permissionsError）
 *    构造三种态，正是为了锁死这一点。
 *
 * 行操作走 ActionButtons：3 个 action（编辑/启停/删除）≥ threshold(3) → 折叠进「操作」下拉；
 * 少于一阈值时平铺显示。因此「只授 edit」时行内直接出现「编辑」「停用」两个按钮，
 * 「只授 remove」时行内只出现「删除」——可据此逐按钮验证 permission 字段生效。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor, fireEvent, act } from "@testing-library/react";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { mockApiBatch, setGenericFallback } from "@/test/utils/createApiMock";
import { QueryClient } from "@tanstack/react-query";
import { useMenuStore } from "@/store/menuStore";
import { PERMISSION_CODES } from "@/constants/permissionCodes";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import RoleManagement from "../index";

const ROLE_ROW = {
  id: "r1",
  roleName: "系统管理员",
  roleKey: "admin",
  roleSort: 1,
  status: 0,
  remark: "",
  createdAt: "2026-01-01 00:00:00",
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

function renderRole() {
  // role 页的 useRoleActions 依赖 react-query 的缓存失效器（useQueryClient）
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithProviders(<RoleManagement />, { queryClient: qc });
}

/** 勾选首行（触发 rowSelection.onChange → selectedRowKeys 非空 → 批量删除入口出现） */
async function selectFirstRow(): Promise<void> {
  const checkboxes = await screen.findAllByRole("checkbox");
  // 第一个是表头「选择全部」，其后是行选择框
  await waitFor(() => expect(checkboxes.length).toBeGreaterThan(0));
  fireEvent.click(checkboxes[checkboxes.length - 1]);
}

describe("角色管理页操作级权限门控（Phase 86 W1）", () => {
  beforeEach(() => {
    mockApiBatch([
      {
        endpoint: "/system/roles/list",
        response: { data: { list: [ROLE_ROW], total: 1, current: 1, pageSize: 10 } },
      },
      {
        endpoint: "/system/roles/statistics",
        response: { data: { total: 1, active: 1, inactive: 0 } },
      },
      { endpoint: "/system/menus/tree-select", response: { data: [] } },
      { endpoint: "/system/departments/tree-select", response: { data: [] } },
    ]);
    setGenericFallback({ data: {} });
  });

  // ==================== A5：无权限不在 DOM ====================

  it("A5-1 只持 list 权限 → 新增/行编辑/行停用/行删除/批量删除均不在 DOM", async () => {
    setPermissionState([PERMISSION_CODES.systemRole.list]);
    renderRole();

    await waitFor(() => {
      expect(screen.getByText("系统管理员")).toBeInTheDocument();
    });

    // 工具栏「新增角色」
    expect(screen.queryByRole("button", { name: /新增角色/ })).not.toBeInTheDocument();
    // 行操作：3 个 action 全部无权限 → ActionButtons 返回 null（连「操作」下拉都不渲染）
    expect(screen.queryByRole("button", { name: "编辑" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "停用" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "删除" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "操作" })).not.toBeInTheDocument();

    // 批量删除（选中行后仍不出现 —— 只写 disabled 会让按钮留在 DOM，这里必须不渲染）
    await selectFirstRow();
    await waitFor(() => {
      expect(screen.queryByRole("button", { name: /批量删除/ })).not.toBeInTheDocument();
    });
  });

  it("A5-2 只持 remove 权限 → 行内只出现「删除」，编辑/停用不在 DOM", async () => {
    setPermissionState([PERMISSION_CODES.systemRole.list, PERMISSION_CODES.systemRole.remove]);
    renderRole();

    expect(await screen.findByRole("button", { name: "删除" })).toBeInTheDocument();
    // 无 edit → 编辑 / 停用（启停归 system:role:edit）不可见
    expect(screen.queryByRole("button", { name: "编辑" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "停用" })).not.toBeInTheDocument();
    // 无 add → 新增不可见
    expect(screen.queryByRole("button", { name: /新增角色/ })).not.toBeInTheDocument();
  });

  // ==================== A6：未就绪 disabled + 提示 ====================

  it("A6-1 权限未就绪（loading）→ 入口仍在 DOM 且 disabled 并带提示", async () => {
    // 注意：permissions 为空但 permissionsLoaded=false —— 这正是「未就绪」态
    setPermissionState([], { loaded: false });
    renderRole();

    const addBtn = await screen.findByRole("button", { name: /新增角色/ });
    expect(addBtn).toBeInTheDocument();
    expect(addBtn).toBeDisabled();
    expect(addBtn.getAttribute("title") ?? "").toContain("权限");

    // 行操作入口：未就绪 → 全部 action 可见但禁用（ActionButtons 内建三态门控）
    const opBtn = await screen.findByRole("button", { name: "操作" });
    expect(opBtn).toBeInTheDocument();
  });

  it("A6-2 权限接口失败（error）→ 入口仍在 DOM 且 disabled 并带提示", async () => {
    setPermissionState([], { loaded: false, error: new Error("权限接口失败") });
    renderRole();

    const addBtn = await screen.findByRole("button", { name: /新增角色/ });
    expect(addBtn).toBeInTheDocument();
    expect(addBtn).toBeDisabled();
    expect(addBtn.getAttribute("title") ?? "").toContain("权限");
  });

  // ==================== A7：有权限可点 ====================

  it("A7-1 持 add + edit 权限 → 新增与行编辑/启停存在且未禁用", async () => {
    setPermissionState([
      PERMISSION_CODES.systemRole.list,
      PERMISSION_CODES.systemRole.add,
      PERMISSION_CODES.systemRole.edit,
    ]);
    renderRole();

    const addBtn = await screen.findByRole("button", { name: /新增角色/ });
    expect(addBtn).toBeInTheDocument();
    expect(addBtn).not.toBeDisabled();

    // 只授 edit（无 remove）→ 行内平铺「编辑」「停用」两个按钮
    const editBtn = await screen.findByRole("button", { name: "编辑" });
    expect(editBtn).not.toBeDisabled();
    const toggleBtn = screen.getByRole("button", { name: "停用" });
    expect(toggleBtn).not.toBeDisabled();
    // 未授 remove → 删除不可见
    expect(screen.queryByRole("button", { name: "删除" })).not.toBeInTheDocument();
  });

  it("A7-2 持 remove 权限并选中行 → 批量删除存在且未禁用", async () => {
    setPermissionState([
      PERMISSION_CODES.systemRole.list,
      PERMISSION_CODES.systemRole.add,
      PERMISSION_CODES.systemRole.remove,
    ]);
    renderRole();

    await selectFirstRow();
    const batchBtn = await screen.findByRole("button", { name: /批量删除/ });
    expect(batchBtn).toBeInTheDocument();
    expect(batchBtn).not.toBeDisabled();
  });

  // ==================== 第四路径：ready 空权限集 vs 未就绪可区分 ====================

  it("A8-1 权限就绪但为空权限集 → 新增不在 DOM（与「未就绪」可区分）", async () => {
    // 空权限集 + 已就绪 = 用户确实没有任何权限 → 隐藏（而非禁用）
    setPermissionState([], { loaded: true });
    renderRole();

    await waitFor(() => {
      expect(screen.getByText("系统管理员")).toBeInTheDocument();
    });
    expect(screen.queryByRole("button", { name: /新增角色/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "操作" })).not.toBeInTheDocument();

    // 复核：同一权限集切到「未就绪」后入口重新出现（证明二者确实不同态）
    act(() => {
      setPermissionState([], { loaded: false });
    });
    const addBtn = await screen.findByRole("button", { name: /新增角色/ });
    expect(addBtn).toBeDisabled();
  });
  // 整页渲染 + antd Table 在并行跑全量用例时较慢，放宽到 30s（与 system/user 同口径）。
}, 30000);
