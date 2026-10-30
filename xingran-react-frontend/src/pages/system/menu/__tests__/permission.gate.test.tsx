/**
 * Phase 86 W1（RBAC-02）—— 菜单管理页操作级权限门控测试。
 *
 * 覆盖三条路径（+ ready 空权限集第四路径）+ `<Switch>` 形态：
 *   A5 无权限不在 DOM：只持 system:menu:list → 新增 / 行编辑 / 行删除 / 批量删除均不存在；
 *   A6 未就绪 disabled：permissionsLoaded=false → 入口仍在 DOM 且 disabled + 带提示（title）；
 *   A7 有权限可点：持 add/edit/remove → 对应入口存在且未禁用；
 *   A9 Switch 三态：弹窗内「显示状态 / 菜单状态」两个 Switch —— ready 无权限 → 整个 Switch
 *      不渲染（不是只禁用）；未就绪 → 渲染但 disabled + 提示。
 *
 * ⚠️ 禁止用 `permissions.length === 0` 判「未就绪」——空权限集与未加载完在数据上
 *    不可区分。本测试用 menuStore 的显式就绪位（permissionsLoaded / permissionsError）
 *    构造三种态，正是为了锁死这一点。
 *
 * ⚠️ 行操作是裸 `<Button icon={…}>编辑/删除</Button>`（非 ActionButtons）——antd 图标自带
 *    aria-label（"edit"/"delete"），按钮的可访问名是「图标名 + 文本」，故用正则匹配。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor, fireEvent, act } from "@testing-library/react";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { mockApiBatch, setGenericFallback } from "@/test/utils/createApiMock";
import { useMenuStore } from "@/store/menuStore";
import { PERMISSION_CODES } from "@/constants/permissionCodes";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import MenuManagement from "../index";

const MENU_ROW = {
  id: "m1",
  menuName: "系统管理",
  parentId: null,
  orderNum: 1,
  path: "/system",
  component: "Layout",
  menuType: "M" as const,
  visible: 1 as const,
  status: 0 as const,
  perms: "",
  icon: null,
  remark: "",
  createTime: "2026-01-01 00:00:00",
  updateTime: "2026-01-01 00:00:00",
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

function renderMenu() {
  return renderWithProviders(<MenuManagement />);
}

/** 勾选首行（触发 rowSelection.onChange → selectedRowKeys 非空 → 批量删除入口出现） */
async function selectFirstRow(): Promise<void> {
  const checkboxes = await screen.findAllByRole("checkbox");
  await waitFor(() => expect(checkboxes.length).toBeGreaterThan(0));
  fireEvent.click(checkboxes[checkboxes.length - 1]);
}

describe("菜单管理页操作级权限门控（Phase 86 W1）", () => {
  beforeEach(() => {
    mockApiBatch([
      {
        endpoint: "/system/menus/tree",
        response: { data: [MENU_ROW] },
      },
    ]);
    setGenericFallback({ data: {} });
  });

  // ==================== A5：无权限不在 DOM ====================

  it("A5-1 只持 list 权限 → 新增/行编辑/行删除/批量删除均不在 DOM", async () => {
    setPermissionState([PERMISSION_CODES.systemMenu.list]);
    renderMenu();

    await waitFor(() => {
      expect(screen.getByText("系统管理")).toBeInTheDocument();
    });

    // 工具栏「新增菜单」
    expect(screen.queryByRole("button", { name: /新增菜单/ })).not.toBeInTheDocument();
    // 行操作（裸 Button，非 ActionButtons → 页面层条件渲染）
    expect(screen.queryByRole("button", { name: /编辑/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /删除/ })).not.toBeInTheDocument();
    // 弹窗打不开 ⇒ 弹窗内两个 Switch 也不在 DOM
    expect(screen.queryAllByRole("switch")).toHaveLength(0);

    // 批量删除（选中行后仍不出现 —— 只写 disabled 会让按钮留在 DOM，这里必须不渲染）
    await selectFirstRow();
    await waitFor(() => {
      expect(screen.queryByRole("button", { name: /批量删除/ })).not.toBeInTheDocument();
    });
  });

  it("A5-2 只持 remove 权限 → 行内只出现「删除」，编辑不在 DOM", async () => {
    setPermissionState([PERMISSION_CODES.systemMenu.list, PERMISSION_CODES.systemMenu.remove]);
    renderMenu();

    expect(await screen.findByRole("button", { name: /删除/ })).toBeInTheDocument();
    // 无 edit → 编辑不可见
    expect(screen.queryByRole("button", { name: /编辑/ })).not.toBeInTheDocument();
    // 无 add → 新增不可见
    expect(screen.queryByRole("button", { name: /新增菜单/ })).not.toBeInTheDocument();
  });

  // ==================== A6：未就绪 disabled + 提示 ====================

  it("A6-1 权限未就绪（loading）→ 入口仍在 DOM 且 disabled 并带提示", async () => {
    // 注意：permissions 为空但 permissionsLoaded=false —— 这正是「未就绪」态
    setPermissionState([], { loaded: false });
    renderMenu();

    const addBtn = await screen.findByRole("button", { name: /新增菜单/ });
    expect(addBtn).toBeInTheDocument();
    expect(addBtn).toBeDisabled();
    expect(addBtn.getAttribute("title") ?? "").toContain("权限");

    const editBtn = await screen.findByRole("button", { name: /编辑/ });
    expect(editBtn).toBeInTheDocument();
    expect(editBtn).toBeDisabled();
    expect(editBtn.getAttribute("title") ?? "").toContain("权限");

    const delBtn = screen.getByRole("button", { name: /删除/ });
    expect(delBtn).toBeDisabled();
    expect(delBtn.getAttribute("title") ?? "").toContain("权限");
  });

  it("A6-2 权限接口失败（error）→ 入口仍在 DOM 且 disabled 并带提示", async () => {
    setPermissionState([], { loaded: false, error: new Error("权限接口失败") });
    renderMenu();

    const addBtn = await screen.findByRole("button", { name: /新增菜单/ });
    expect(addBtn).toBeInTheDocument();
    expect(addBtn).toBeDisabled();
    expect(addBtn.getAttribute("title") ?? "").toContain("权限");
  });

  // ==================== A7：有权限可点 ====================

  it("A7-1 持 add + edit + remove 权限 → 新增/行编辑/行删除存在且未禁用", async () => {
    setPermissionState([
      PERMISSION_CODES.systemMenu.list,
      PERMISSION_CODES.systemMenu.add,
      PERMISSION_CODES.systemMenu.edit,
      PERMISSION_CODES.systemMenu.remove,
    ]);
    renderMenu();

    const addBtn = await screen.findByRole("button", { name: /新增菜单/ });
    expect(addBtn).toBeInTheDocument();
    expect(addBtn).not.toBeDisabled();

    const editBtn = await screen.findByRole("button", { name: /编辑/ });
    expect(editBtn).not.toBeDisabled();

    const delBtn = screen.getByRole("button", { name: /删除/ });
    expect(delBtn).not.toBeDisabled();
  });

  it("A7-2 持 remove 权限并选中行 → 批量删除存在且未禁用", async () => {
    setPermissionState([
      PERMISSION_CODES.systemMenu.list,
      PERMISSION_CODES.systemMenu.add,
      PERMISSION_CODES.systemMenu.remove,
    ]);
    renderMenu();

    await selectFirstRow();
    const batchBtn = await screen.findByRole("button", { name: /批量删除/ });
    expect(batchBtn).toBeInTheDocument();
    expect(batchBtn).not.toBeDisabled();
  });

  // ==================== 第四路径：ready 空权限集 vs 未就绪可区分 ====================

  it("A8-1 权限就绪但为空权限集 → 新增不在 DOM（与「未就绪」可区分）", async () => {
    // 空权限集 + 已就绪 = 用户确实没有任何权限 → 隐藏（而非禁用）
    setPermissionState([], { loaded: true });
    renderMenu();

    await waitFor(() => {
      expect(screen.getByText("系统管理")).toBeInTheDocument();
    });
    expect(screen.queryByRole("button", { name: /新增菜单/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /编辑/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /删除/ })).not.toBeInTheDocument();
  });

  // ==================== A9：`<Switch>` 形态三态 ====================

  it("A9-1 弹窗内两个 Switch：ready 无权限 → 不渲染；未就绪 → 渲染但 disabled + 提示", async () => {
    setPermissionState([
      PERMISSION_CODES.systemMenu.list,
      PERMISSION_CODES.systemMenu.add,
    ]);
    renderMenu();

    const addBtn = await screen.findByRole("button", { name: /新增菜单/ });
    fireEvent.click(addBtn);
    await screen.findByRole("dialog");

    // 有 add 权限 → 弹窗内「显示状态 / 菜单状态」两个 Switch 存在且可用
    expect(screen.getAllByRole("switch")).toHaveLength(2);
    expect(screen.getAllByRole("switch")[0]).not.toBeDisabled();

    // 切到「ready + 空权限集」→ 整个 Switch 不渲染（不是只禁用）
    act(() => {
      setPermissionState([], { loaded: true });
    });
    await waitFor(() => {
      expect(screen.queryAllByRole("switch")).toHaveLength(0);
    });

    // 切到「未就绪」→ 重新渲染但 disabled + 提示
    act(() => {
      setPermissionState([], { loaded: false });
    });
    await waitFor(() => {
      expect(screen.getAllByRole("switch")).toHaveLength(2);
    });
    const sw = screen.getAllByRole("switch")[0];
    expect(sw).toBeDisabled();
    expect(sw.getAttribute("title") ?? "").toContain("权限");
  });
  // 整页渲染 + antd Table/Modal 在并行跑全量用例时较慢，放宽到 30s（与 system/user 同口径）。
}, 30000);
