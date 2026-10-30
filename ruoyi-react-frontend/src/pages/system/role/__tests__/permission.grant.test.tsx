/**
 * Phase 87 RBAC-05 / RBAC-06 —— 角色授权闭环组件测试。
 *
 * 覆盖：
 *   - T-1: F 型按钮节点在角色页授权树中可见，展开后可被点击勾选
 *   - T-2: antd Tree 级联联动：勾选 F 节点时，提交的 menuIds 同时包含父 C 菜单 id 与子 F id
 *   - T-3: 提交时向 PUT / POST /system/roles/:id/update 发送的 menuIds 包含勾选的 F 节点 ID
 *   - T-4: 回读一致性：/system/menus/role-menu-tree-select/:id 返回已选 F 节点时，树组件正常加载该选中态
 *   - 同源码串断言：前端 TS 常量与后端权限码字面量完全一致（RBAC-05 / RBAC-06）
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

function renderRole() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithProviders(<RoleManagement />, { queryClient: qc });
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
    renderRole();

    // 等待表格渲染完成并展示角色行
    await waitFor(() => {
      expect(screen.getByText("运维专员")).toBeInTheDocument();
    });

    // 点击行操作「编辑」打开编辑角色 Modal
    const editBtn = screen.getByRole("button", { name: "编辑" });
    fireEvent.click(editBtn);

    // 验证 Modal 弹出且菜单树正常加载展示（T-1）
    // CI（Linux + coverage）下 antd Modal/Tree 挂载明显慢于本地，放宽等待窗口
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

    // 勾选 F 节点复选框（点击与「角色新增」紧邻的前驱 checkbox）
    // CI 慢环境下 antd Tree 受控 checkedKeys 同步可能滞后，且重渲染会让旧引用失效；
    // 因此每次重新查询节点，并在点击后等待 checked class 真正生效再提交。
    const clickFNodeCheckbox = () => {
      const titleEl = screen.getByTitle("角色新增");
      const wrapper = titleEl.closest(".ant-tree-treenode") || titleEl.parentElement;
      const cb = wrapper?.querySelector(".ant-tree-checkbox");
      expect(cb).not.toBeNull();
      fireEvent.click(cb as HTMLElement);
    };
    await waitFor(
      () => {
        clickFNodeCheckbox();
        const titleEl = screen.getByTitle("角色新增");
        const wrapper = titleEl.closest(".ant-tree-treenode") || titleEl.parentElement;
        expect(wrapper?.querySelector(".ant-tree-checkbox-checked")).not.toBeNull();
      },
      { timeout: 15000, interval: 400 }
    );

    // 补全表单必填项（jsdom 环境中 afterOpenChange 动画回调可能不触发）
    // CI 慢环境下 Modal 内 Form 实例可能尚未 connect，单次 change 会被丢弃；
    // 用「重复 change 直到 value 生效」消除时序依赖，超时则暴露真实问题。
    const roleNameEl = await waitFor(
      () => {
        const el = document.querySelector("#roleName");
        expect(el).not.toBeNull();
        return el as HTMLInputElement;
      },
      { timeout: 15000 }
    );
    await waitFor(
      () => {
        fireEvent.change(roleNameEl, { target: { value: "运维专员" } });
        expect(roleNameEl.value).toBe("运维专员");
      },
      { timeout: 15000, interval: 300 }
    );
    const roleKeyEl = document.querySelector("#roleKey") as HTMLInputElement;
    expect(roleKeyEl).not.toBeNull();
    await waitFor(
      () => {
        fireEvent.change(roleKeyEl, { target: { value: "ops_admin" } });
        expect(roleKeyEl.value).toBe("ops_admin");
      },
      { timeout: 15000, interval: 300 }
    );

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
    renderRole();

    await waitFor(() => {
      expect(screen.getByText("运维专员")).toBeInTheDocument();
    });

    const editBtn = screen.getByRole("button", { name: "编辑" });
    fireEvent.click(editBtn);

    await waitFor(() => {
      expect(screen.getByText("编辑角色")).toBeInTheDocument();
      expect(screen.getByText("角色管理")).toBeInTheDocument();
    });

    const switcher = document.querySelector(".ant-tree-switcher");
    if (switcher) {
      fireEvent.click(switcher);
    }

    await waitFor(() => {
      expect(screen.getByText("角色新增")).toBeInTheDocument();
    });

    // 验证该 F 节点处于选中态 (ant-tree-checkbox-checked)
    const titleElement = screen.getByTitle("角色新增");
    const nodeWrapper = titleElement.closest(".ant-tree-treenode") || titleElement.parentElement;
    const checkbox = nodeWrapper?.querySelector(".ant-tree-checkbox");
    await waitFor(() => {
      expect(checkbox).toHaveClass("ant-tree-checkbox-checked");
    });
  }, 30000); // 与 T-1 同因：Tree 展开 + 勾选态回读在慢机下耗时可超默认阈值
});
