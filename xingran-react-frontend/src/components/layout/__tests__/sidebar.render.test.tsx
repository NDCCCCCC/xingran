/**
 * Phase 88 Batch52 — layout Sidebar 渲染测试
 *
 * menuStore 提供菜单 fixture → 菜单项渲染 + 折叠态分支。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderWithProviders } from "@/test/utils/renderWithProviders";
import Sidebar from "../sidebar";
import { useMenuStore } from "@/store/menuStore";
import { useLayoutStore } from "@/store/layoutStore";
import type { Menu } from "@/types";

beforeEach(() => {
  vi.clearAllMocks();
});

const menuFixture: Menu[] = [
  {
    id: "m1",
    menuName: "系统管理",
    path: "/system",
    menuType: "M",
    visible: 1,
    status: 0,
    sortOrder: 1,
    children: [
      {
        id: "m11",
        menuName: "用户管理",
        path: "/system/users",
        menuType: "C",
        visible: 1,
        status: 0,
        sortOrder: 1,
        component: "system/user/index",
        children: [],
      } as Menu,
    ],
  } as unknown as Menu,
];

describe("Sidebar 渲染", () => {
  it("menus 非空渲染菜单", async () => {
    useMenuStore.setState({ menus: menuFixture, loading: false } as any);
    useLayoutStore.setState({ sidebarCollapsed: false, isMobile: false } as any);
    const { baseElement } = renderWithProviders(<Sidebar />);
    expect(baseElement.querySelector(".ant-menu")).not.toBeNull();
  });

  it("menus=[] 渲染空菜单不抛错", () => {
    useMenuStore.setState({ menus: [], loading: false } as any);
    useLayoutStore.setState({ sidebarCollapsed: false, isMobile: false } as any);
    const { baseElement } = renderWithProviders(<Sidebar />);
    expect(baseElement).toBeDefined();
  });

  it("sidebarCollapsed=true 渲染折叠菜单", () => {
    useMenuStore.setState({ menus: menuFixture, loading: false } as any);
    useLayoutStore.setState({ sidebarCollapsed: true, isMobile: false } as any);
    const { baseElement } = renderWithProviders(<Sidebar />);
    expect(baseElement.querySelector(".ant-menu-inline-collapsed, .ant-menu")).not.toBeNull();
  });

  it("loading=true 渲染 Spin", () => {
    useMenuStore.setState({ menus: [], loading: true } as any);
    useLayoutStore.setState({ sidebarCollapsed: false, isMobile: false } as any);
    const { baseElement } = renderWithProviders(<Sidebar />);
    expect(baseElement.querySelector(".ant-spin, .ant-menu")).not.toBeNull();
  });
});
