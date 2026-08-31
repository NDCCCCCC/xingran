/**
 * Phase 88 Batch271 — types/menu 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import type { RouteMeta, MenuRouteConfig, BreadcrumbItem, RoutePermissionCheck } from "../menu";

describe("types/menu", () => {
  it("RouteMeta 必填 + 可选", () => {
    const m: RouteMeta = { title: "用户管理" };
    expect(m.title).toBe("用户管理");
  });

  it("RouteMeta 完整字段", () => {
    const m: RouteMeta = {
      title: "系统",
      icon: "Setting",
      hidden: false,
      affix: true,
      keepAlive: true,
      permissions: ["system:user:list"],
      roles: ["admin"],
      i18nKey: "system",
      noCache: false,
      link: "https://example.com",
    };
    expect(m.permissions?.length).toBe(1);
  });

  it("MenuRouteConfig shape", () => {
    const c: MenuRouteConfig = {
      path: "user",
      component: "pages/user",
      meta: { title: "用户" },
    };
    expect(c.path).toBe("user");
  });

  it("MenuRouteConfig 含 children + redirect", () => {
    const c: MenuRouteConfig = {
      path: "system",
      component: "layout",
      meta: { title: "系统" },
      redirect: "/system/user",
      children: [
        {
          path: "user",
          component: "pages/user",
          meta: { title: "用户" },
        },
      ],
    };
    expect(c.children?.length).toBe(1);
  });

  it("BreadcrumbItem shape", () => {
    const b: BreadcrumbItem = { path: "/user", title: "用户" };
    expect(b.path).toBe("/user");
  });

  it("RoutePermissionCheck shape", () => {
    const c: RoutePermissionCheck = {
      hasPermission: true,
      missingPermissions: ["system:user:delete"],
    };
    expect(c.missingPermissions?.length).toBe(1);
  });
});
