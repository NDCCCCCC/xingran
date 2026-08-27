/**
 * Phase 84 84-02a — Sidebar utils + helper 测试
 */
import { describe, it, expect } from "vitest";
import { buildFullPath, findMenuById, findMenuByFullPath } from "../sidebar.utils";
import { buildMenuPathMap } from "../sidebar-helper";
import type { Menu as MenuType } from "@/types";

const sampleMenu = (overrides: Partial<MenuType> = {}): MenuType => ({
  id: 1,
  parentId: null,
  path: "users",
  component: null,
  menuType: "M",
  icon: null,
  sortOrder: 0,
  title: "用户管理",
  visible: 1,
  ...overrides,
});

describe("buildFullPath", () => {
  it("returns absolute path when path starts with /", () => {
    expect(buildFullPath(sampleMenu({ path: "/users" }))).toBe("/users");
  });

  it("joins parent path and child path with slash", () => {
    expect(buildFullPath(sampleMenu({ path: "users" }), "/admin")).toBe("/admin/users");
  });

  it("returns /path when no parent path", () => {
    expect(buildFullPath(sampleMenu({ path: "dashboard" }))).toBe("/dashboard");
  });

  it("returns parentPath when child path is empty", () => {
    expect(buildFullPath(sampleMenu({ path: "" }), "/admin")).toBe("/admin");
  });
});

describe("findMenuById", () => {
  const menus: MenuType[] = [
    sampleMenu({ id: 1, path: "users", title: "Users" }),
    sampleMenu({ id: 2, path: "roles", title: "Roles", parentId: 1 }),
  ];

  it("findMenuById is callable", () => {
    const found = findMenuById(menus, "1");
  });

  it("returns null for missing id", () => {
    expect(findMenuById(menus, "999")).toBeNull();
  });
});

describe("findMenuByFullPath", () => {
  const menus: MenuType[] = [sampleMenu({ id: 1, path: "users", title: "Users" })];

  it("finds menu by path", () => {
    expect(findMenuByFullPath(menus, "users")).not.toBeNull();
  });

  it("returns null for unknown path", () => {
    expect(findMenuByFullPath(menus, "non-existent")).toBeNull();
  });
});

describe("buildMenuPathMap", () => {
  it("builds path map for flat menu list", () => {
    const menus: MenuType[] = [
      sampleMenu({ id: 1, path: "users" }),
      sampleMenu({ id: 2, path: "roles" }),
    ];
    const map = buildMenuPathMap(menus);
    expect(map).toBeInstanceOf(Map);
  });
});
