/**
 * v1.24 Phase 78（#16）—— 系统用户页操作级权限门控测试（PERM-03）。
 *
 * 覆盖验收项：
 *   A5 权限码门控：mock permissions=['system:user:list'] → 编辑/重置密码/删除按钮**不存在**；
 *                   含 system:user:edit → 编辑按钮存在；含 remove → 删除按钮存在。
 *   A6 三态门控：权限**未就绪**（permissionsLoaded=false / permissionsError 非空）
 *                   → 按钮**渲染但 disabled**，且带提示（title）。
 *
 * ⚠️ 禁止用 `permissions.length === 0` 判「未就绪」——空权限集与未加载完在数据上
 *    不可区分。本测试用 menuStore 的显式就绪位（permissionsLoaded / permissionsError）
 *    构造三种态，正是为了锁死这一点。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { mockApiBatch, setGenericFallback } from "@/test/utils/createApiMock";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useMenuStore } from "@/store/menuStore";
import { PERMISSION_CODES } from "@/constants/permissionCodes";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import UserManagement from "../index";

const USER_ROW = {
  id: "u1",
  username: "admin",
  nickname: "管理员",
  status: 0,
  deptName: "运维部",
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

function renderUser() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithProviders(
    <QueryClientProvider client={qc}>
      <UserManagement />
    </QueryClientProvider>
  );
}

describe("系统用户页操作级权限门控（v1.24 #16）", () => {  beforeEach(() => {
    mockApiBatch([
      {
        endpoint: "/system/users/list",
        response: { data: { list: [USER_ROW], total: 1, current: 1, pageSize: 10 } },
      },
      { endpoint: "/system/users/statistics", response: { data: { total: 1, active: 1, inactive: 0 } } },
      { endpoint: "/system/roles/all", response: { data: [] } },
      { endpoint: "/system/posts/list", response: { data: { list: [], total: 0 } } },
      // 部门树（useDeptTree → useMemo 统计部门数，要求数组）
      { endpoint: "/system/departments/tree", response: { data: [] } },
      // 经理预览（页面渲染时批量拉取，要求数组）
      { endpoint: "/system/manager-rules/manager-preview", response: { data: [] } },
    ]);
    setGenericFallback({ data: {} });
  });

  // ==================== A5：权限码门控 ====================

  it("A5-1 只持 list 权限 → 编辑/重置密码/删除/新增/导入按钮均不存在", async () => {
    setPermissionState([PERMISSION_CODES.systemUser.list]);
    renderUser();

    await waitFor(() => {
      expect(screen.getByText("admin")).toBeInTheDocument();
    });

    expect(screen.queryByRole("button", { name: "编辑" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "重置密码" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "删除" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /新增用户/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /导入用户/ })).not.toBeInTheDocument();
  });

  it("A5-2 含 edit 权限 → 编辑按钮存在", async () => {
    setPermissionState([PERMISSION_CODES.systemUser.list, PERMISSION_CODES.systemUser.edit]);
    renderUser();

    expect(await screen.findByRole("button", { name: "编辑" })).toBeInTheDocument();
    // 未授予 remove → 删除仍不可见
    expect(screen.queryByRole("button", { name: "删除" })).not.toBeInTheDocument();
  });

  it("A5-3 含 remove 权限 → 删除按钮存在", async () => {
    setPermissionState([PERMISSION_CODES.systemUser.list, PERMISSION_CODES.systemUser.remove]);
    renderUser();

    expect(await screen.findByRole("button", { name: "删除" })).toBeInTheDocument();
    // 未授予 resetPwd → 重置密码不可见
    expect(screen.queryByRole("button", { name: "重置密码" })).not.toBeInTheDocument();
  });

  it("A5-4 含 add/import 权限 → 新增/导入按钮存在", async () => {
    setPermissionState([
      PERMISSION_CODES.systemUser.list,
      PERMISSION_CODES.systemUser.add,
      PERMISSION_CODES.systemUser.import,
    ]);
    renderUser();

    expect(await screen.findByRole("button", { name: /新增用户/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /导入用户/ })).toBeInTheDocument();
  });

  // ==================== A6：三态门控 ====================

  it("A6-1 权限未就绪（loading）→ 按钮渲染但 disabled 且有提示", async () => {
    // 注意：permissions 为空但 permissionsLoaded=false —— 这正是「未就绪」态
    setPermissionState([], { loaded: false });
    renderUser();

    const addBtn = await screen.findByRole("button", { name: /新增用户/ });
    expect(addBtn).toBeInTheDocument();
    expect(addBtn).toBeDisabled();
    expect(addBtn.getAttribute("title") ?? "").toContain("权限");
  });

  it("A6-2 权限接口失败（error）→ 按钮渲染但 disabled 且有提示", async () => {
    setPermissionState([], { loaded: false, error: new Error("权限接口失败") });
    renderUser();

    const addBtn = await screen.findByRole("button", { name: /新增用户/ });
    expect(addBtn).toBeInTheDocument();
    expect(addBtn).toBeDisabled();
    expect(addBtn.getAttribute("title") ?? "").toContain("权限");
  });

  it("A6-3 权限就绪但为空权限集 → 按钮不渲染（与「未就绪」可区分）", async () => {
    // 空权限集 + 已就绪 = 用户确实没有任何权限 → 隐藏（而非禁用）
    setPermissionState([], { loaded: true });
    renderUser();

    await waitFor(() => {
      expect(screen.getByText("admin")).toBeInTheDocument();
    });
    expect(screen.queryByRole("button", { name: /新增用户/ })).not.toBeInTheDocument();
  });
  // 整页渲染 + antd Table 在并行跑全量用例时较慢（本文件 7 个用例），
  // 放宽到 30s（与 pages/network/ports、asset/reconciliation 同口径）。
}, 30000);
