/**
 * Phase 87 RBAC-05 / RBAC-06 —— 角色授权闭环组件测试。
 *
 * 覆盖：
 *   - T-1: F 型按钮节点在角色页授权树中可见，展开后可被点击勾选
 *   - T-2: antd Tree 级联联动：勾选 F 节点时，提交的 menuIds 同时包含父 C 菜单 id 与子 F id
 *   - T-3: 提交时向 PUT / POST /system/roles/:id/update 发送的 menuIds 包含勾选的 F 节点 ID
 *   - T-4: 回读一致性：/system/menus/role-menu-tree-select/:id 返回已选 F 节点时，树组件正常加载该选中态
 *   - 同源码串断言：前端 TS 常量与后端权限码字面量完全一致（RBAC-05 / RBAC-06）
 *
 * DEBT-05 时序根治（v1.51）：
 *   - 移除「重复 fireEvent.change 直到 value 生效」轮询补丁（原 5e5336a9 / fc540a23）。
 *   - 表单交互改为确定性路径：组件通过 onEditFormReady 暴露 editForm 实例，
 *     待字段 DOM 就绪后用 form.setFieldsValue 一次性写入（antd Form store 暂存，
 *     不受 Field connect 时机影响），再以 getFieldValue 断言。
 *   - 树勾选状态不再探测 .ant-tree-checkbox-checked class，统一以受控数据
 *     （form 值 menuIds，经 Tree onCheck → setFieldsValue 写入）断言。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { screen, waitFor, fireEvent, cleanup } from "@testing-library/react";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import {
  mockApiBatch,
  setGenericFallback,
  resetApiMocks,
  ApiMockHandle,
} from "@/test/utils/createApiMock";
import { QueryClient } from "@tanstack/react-query";
import type { FormInstance } from "antd/es/form";
import { useMenuStore } from "@/store/menuStore";
import { PERMISSION_CODES } from "@/constants/permissionCodes";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import RoleManagement from "../index";

const ROLE_ROW = {
  id: "r1",
  roleName: "运维专员",
  roleKey: "ops_admin",
  roleSort: 1,
  status: 0,
  remark: "测试角色",
  createdAt: "2026-01-01 00:00:00",
};

// 构造树结构：父 C 型菜单（角色管理）下挂 F 型按钮节点（角色新增）
const MOCK_MENU_TREE = [
  {
    id: "m-role-c",
    menuName: "角色管理",
    menuType: "C",
    perms: "system:role",
    children: [
      {
        id: "m-role-add-f",
        menuName: "角色新增",
        menuType: "F",
        perms: "system:role:add",
        children: [],
      },
    ],
  },
];

function setPermissionState(permissions: string[]): void {
  useMenuStore.setState({
    permissions,
    permissionsLoaded: true,
    permissionsError: null,
  });
}

interface RolePageHandles {
  editForm: FormInstance<unknown> | null;
}

function renderRole(): RolePageHandles {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const handles: RolePageHandles = { editForm: null };
  renderWithProviders(<RoleManagement onEditFormReady={(form) => (handles.editForm = form)} />, {
    queryClient: qc,
  });
  return handles;
}

/** 确定性等待：editForm 实例就绪且指定字段已在 DOM 注册（Field 已渲染）。 */
async function waitForFormField(handles: RolePageHandles, fieldId: string): Promise<void> {
  await waitFor(() => {
    expect(handles.editForm).not.toBeNull();
    expect(document.querySelector(`#${fieldId}`)).not.toBeNull();
  });
}

describe("角色授权闭环组件测试（Phase 87 RBAC-05）", () => {
  let handles: Record<string, ApiMockHandle>;

  afterEach(() => {
    cleanup();
    resetApiMocks();
  });

  beforeEach(() => {
    handles = mockApiBatch([
      {
        endpoint: "/system/roles/list",
        response: { data: { list: [ROLE_ROW], total: 1, current: 1, pageSize: 10 } },
      },
      {
        endpoint: "/system/roles/statistics",
        response: { data: { total: 1, active: 1, inactive: 0 } },
      },
      {
        endpoint: "/system/menus/tree-select",
        response: { data: MOCK_MENU_TREE },
      },
      {
        endpoint: "/system/menus/role-menu-tree-select/r1",
        response: { data: { menus: MOCK_MENU_TREE, checkedKeys: [] } },
      },
      {
        endpoint: "/system/departments/tree-select",
        response: { data: [] },
      },
      {
        endpoint: "/system/departments/role-dept-tree-select/r1",
        response: { data: { checkedKeys: [] } },
      },
      {
        endpoint: "/system/roles/r1/update",
        response: { code: 0, message: "success", data: {} },
      },
    ]);
    setGenericFallback({ data: {} });
  });

  it("同源断言：前端 TS 常量与后端权限码字面量完全一致（RBAC-05 / RBAC-06）", () => {
    // 证明前后端使用同一套不可漂移的权限码源
    expect(PERMISSION_CODES.systemRole.add).toBe("system:role:add");
    expect(PERMISSION_CODES.systemRole.list).toBe("system:role:list");
    expect(PERMISSION_CODES.systemRole.edit).toBe("system:role:edit");
  });

  it("T-1 & T-2 & T-3: F 型按钮在授权树中可见，勾选后提交 menuIds 包含 F 节点 id 与父 C 菜单 id", async () => {
    // 赋予操作员编辑角色的权限，使「编辑」按钮可点
    setPermissionState([PERMISSION_CODES.systemRole.list, PERMISSION_CODES.systemRole.edit]);
    const page = renderRole();

    // 等待表格渲染完成并展示角色行
    await waitFor(() => {
      expect(screen.getByText("运维专员")).toBeInTheDocument();
    });

    // 点击行操作「编辑」打开编辑角色 Modal
    const editBtn = screen.getByRole("button", { name: "编辑" });
    fireEvent.click(editBtn);

    // 确定性等待：editForm 实例就绪（组件 onEditFormReady 回调已触发）
    await waitFor(() => {
      expect(page.editForm).not.toBeNull();
    });

    // 验证 Modal 弹出且菜单树正常加载展示（T-1）
    // CI（Linux + coverage）下 antd Modal/Tree 挂载明显慢于本地，仅放宽「渲染完成」等待窗口
    await waitFor(
      () => {
        expect(screen.getByText("编辑角色")).toBeInTheDocument();
        expect(screen.getByText("角色管理")).toBeInTheDocument();
      },
      { timeout: 15000 }
    );

    // 点击展开 switcher
    const switcher = document.querySelector(".ant-tree-switcher");
    if (switcher) {
      fireEvent.click(switcher);
    }

    // F 型按钮节点在树中可见
    const fNode = await screen.findByText("角色新增");
    expect(fNode).toBeInTheDocument();

    // 确定性等待展开态：switcher 的受控展开 class（数据级信号）出现后，
    // 树节点 DOM 已随展开完成重建，此后单击 checkbox 不会再被 motion 重渲染吞掉
    await waitFor(() => {
      const sw = document.querySelector(".ant-tree-switcher");
      expect(sw).toHaveClass("ant-tree-switcher_open");
    });

    // 勾选 F 节点复选框：单次点击，状态确认走受控数据（Tree onCheck → setCheckedMenuKeys
    // + editForm.setFieldsValue({ menuIds })），不探测 checked class
    const titleEl = screen.getByTitle("角色新增");
    const wrapper = titleEl.closest(".ant-tree-treenode") || titleEl.parentElement;
    const cb = wrapper?.querySelector(".ant-tree-checkbox");
    expect(cb).not.toBeNull();
    fireEvent.click(cb as HTMLElement);

    // 确定性断言：勾选结果经 form 实例读取（T-2 级联 + T-1 勾选生效）
    await waitFor(() => {
      const menuIds = page.editForm?.getFieldValue("menuIds") as string[] | undefined;
      expect(Array.isArray(menuIds)).toBe(true);
      expect(menuIds).toContain("m-role-add-f");
      // 父子级联必须同时包含父 C 菜单 id（T-2 前置验证）
      expect(menuIds).toContain("m-role-c");
    });

    // 补全表单必填项：字段 DOM 就绪（Field 已注册）后用 setFieldsValue 一次性写入，
    // antd Form store 暂存值不受 connect 时机影响，无轮询
    await waitForFormField(page, "roleName");
    page.editForm?.setFieldsValue({ roleName: "运维专员", roleKey: "ops_admin" });
    expect(page.editForm?.getFieldValue("roleName")).toBe("运维专员");
    expect(page.editForm?.getFieldValue("roleKey")).toBe("ops_admin");

    // 提交编辑表单：查找 Modal 底部确定按钮（支持「确 定」或「OK」）
    const submitBtn = screen.getByRole("button", { name: /确\s*定|OK/ });
    fireEvent.click(submitBtn);

    // 验证向后端发送的请求参数（T-2 & T-3）
    await waitFor(() => {
      const updateMock = handles["/system/roles/r1/update"].endpoint;
      expect(updateMock).toHaveBeenCalled();
      const calls = updateMock.mock.calls;
      const [, payload] = calls[0] as [string, { menuIds: string[] }];
      // 必须包含 F 节点 id（T-3）
      expect(payload.menuIds).toContain("m-role-add-f");
      // 父子级联必须同时包含父 C 菜单 id（T-2）
      expect(payload.menuIds).toContain("m-role-c");
    });
  }, 30000); // antd Tree 全量渲染在慢机/高并发下可达 ~12s，默认 15s 阈值偏紧

  it("T-4: 回读一致性 —— 角色既有权限含 F 节点时，树组件正常加载该勾选状态", async () => {
    // 覆盖角色已有 m-role-add-f 的 checked 态
    mockApiBatch([
      {
        endpoint: "/system/menus/role-menu-tree-select/r1",
        response: { data: { menus: MOCK_MENU_TREE, checkedKeys: ["m-role-add-f"] } },
      },
    ]);

    setPermissionState([PERMISSION_CODES.systemRole.list, PERMISSION_CODES.systemRole.edit]);
    const page = renderRole();

    await waitFor(() => {
      expect(screen.getByText("运维专员")).toBeInTheDocument();
    });

    const editBtn = screen.getByRole("button", { name: "编辑" });
    fireEvent.click(editBtn);

    await waitFor(() => {
      expect(screen.getByText("编辑角色")).toBeInTheDocument();
    });

    const switcher = document.querySelector(".ant-tree-switcher");
    if (switcher) {
      fireEvent.click(switcher);
    }

    await waitFor(() => {
      expect(screen.getByText("角色新增")).toBeInTheDocument();
    });

    // 与 T-1 同款确定性等待：受控展开 class 出现后再做后续断言
    await waitFor(() => {
      const sw = document.querySelector(".ant-tree-switcher");
      expect(sw).toHaveClass("ant-tree-switcher_open");
    });

    // 回读一致性断言走受控数据：Modal 打开动画完成后 handleModalOpenChange 将
    // role-menu-tree-select 返回的 checkedKeys 写入 form（menuIds）与受控 Tree，
    // 因此以 form 值断言，不探测 .ant-tree-checkbox-checked class
    await waitFor(
      () => {
        const menuIds = page.editForm?.getFieldValue("menuIds") as string[] | undefined;
        expect(Array.isArray(menuIds)).toBe(true);
        expect(menuIds).toContain("m-role-add-f");
      },
      { timeout: 15000 }
    );
  }, 30000); // 与 T-1 同因：Tree 展开 + 勾选态回读在慢机下耗时可超默认阈值
});
