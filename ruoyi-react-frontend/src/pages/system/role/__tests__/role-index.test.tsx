/**
 * Phase 88 Batch21c — system/role + system/post 主页
 * mock 字段 camelCase 匹配 dataIndex(roleName/roleKey/roleSort)。
 */
import { describe, it, expect, vi } from "vitest";
import { renderPageWithEndpoints } from "@/test/utils/renderPage";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import RoleManagement from "../index";
import PostManagement from "@/pages/system/post/index";

async function waitText(text: string, timeoutMs = 6000): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (document.body.innerHTML.includes(text)) return true;
    await new Promise((r) => setTimeout(r, 150));
  }
  return false;
}

const rolesList = {
  data: {
    list: [
      {
        id: "r1",
        roleName: "系统管理员",
        roleKey: "admin",
        roleSort: 1,
        status: 0,
        createdAt: "2026-01-01",
      },
    ],
    total: 1,
    current: 1,
    pageSize: 10,
  },
};

const postsList = {
  data: {
    list: [
      {
        id: "p1",
        postCode: "OPS",
        postName: "运维工程师",
        postSort: 1,
        status: 0,
        createdAt: "2026-01-01",
      },
    ],
    total: 1,
    current: 1,
    pageSize: 10,
  },
};

describe("system/role — index", () => {
  it("renders role rows", async () => {
    await renderPageWithEndpoints(<RoleManagement />, {
      endpoints: {
        "/system/roles/list": rolesList,
        "/system/menus/role-menu-tree-select/r1": { data: { menus: [], checkedKeys: [] } },
        "/system/departments/role-dept-tree-select/r1": { data: { depts: [], checkedKeys: [] } },
      },
    });
    expect(await waitText("系统管理员")).toBe(true);
  });

  it("renders empty roles state", async () => {
    await renderPageWithEndpoints(<RoleManagement />, {
      endpoints: {
        "/system/roles/list": { data: { list: [], total: 0, current: 1, pageSize: 10 } },
      },
    });
    const ok = await waitText("No data");
    expect(ok || document.querySelector(".ant-table-placeholder")).toBeTruthy();
  });
});

describe("system/post — index", () => {
  it("renders post rows", async () => {
    await renderPageWithEndpoints(<PostManagement />, {
      endpoints: {
        "/system/posts/list": postsList,
      },
    });
    expect(await waitText("运维工程师")).toBe(true);
  });

  it("renders empty posts state", async () => {
    await renderPageWithEndpoints(<PostManagement />, {
      endpoints: {
        "/system/posts/list": { data: { list: [], total: 0, current: 1, pageSize: 10 } },
      },
    });
    const ok = await waitText("No data");
    expect(ok || document.querySelector(".ant-table-placeholder")).toBeTruthy();
  });
});
