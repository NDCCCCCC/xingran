/**
 * Phase 88 Batch34 — router/routeConfigManager 单元测试
 *
 * singleton(全 module 共享 state)用 beforeEach clear,避免全局副作用。
 */
import { describe, it, expect, beforeEach } from "vitest";
import { routeConfigManager } from "../routeConfigManager";
import type { Menu } from "@/types";

function makeMenu(overrides: Partial<Menu>): Menu {
  return {
    id: 1,
    parentId: 0,
    menuName: "默认",
    menuType: "M",
    path: "",
    component: "",
    icon: "",
    sortOrder: 0,
    visible: 1,
    ...overrides,
  } as Menu;
}

function seed() {
  routeConfigManager.clear();
  routeConfigManager.initialize([
    makeMenu({
      id: 1,
      menuName: "首页",
      path: "dashboard",
      component: "pages/dashboard/index",
      visible: 1,
      meta: { title: "仪表盘" } as any,
    }),
    makeMenu({
      id: 2,
      menuName: "用户管理",
      path: "user",
      component: "pages/system/user/index",
      visible: 1,
      meta: { title: "用户管理", permissions: ["system:user:list"] } as any,
    }),
    makeMenu({
      id: 3,
      menuName: "隐藏菜单",
      path: "hidden",
      component: "pages/x/index",
      visible: 0,
      meta: { title: "隐藏" } as any,
    }),
    makeMenu({
      id: 4,
      menuName: "按钮",
      path: "btn",
      component: "",
      menuType: "F",
      visible: 1,
      meta: { title: "按钮" } as any,
    }),
    makeMenu({
      id: 5,
      menuName: "嵌套",
      path: "sub",
      component: "pages/sub/index",
      visible: 1,
      meta: { title: "嵌套父" } as any,
      children: [
        makeMenu({
          id: 6,
          menuName: "嵌套子",
          path: "child",
          component: "pages/sub/child/index",
          visible: 1,
          meta: { title: "嵌套子" } as any,
        }),
      ],
    }),
  ] as Menu[]);
}

describe("routeConfigManager 单例行为", () => {
  beforeEach(seed);

  it("initialize 后 isInitialized=true", () => {
    expect(routeConfigManager.isInitialized()).toBe(true);
  });

  it("clear 后 isInitialized=false", () => {
    routeConfigManager.clear();
    expect(routeConfigManager.isInitialized()).toBe(false);
  });

  it("getAllRoutes 返回全部解析后路由", () => {
    const all = routeConfigManager.getAllRoutes();
    expect(all.length).toBeGreaterThanOrEqual(4);
    const paths = all.map((r) => r.path);
    expect(paths).toContain("dashboard");
    expect(paths).toContain("user");
  });

  it("按钮(menuType=F)被跳过", () => {
    const all = routeConfigManager.getAllRoutes();
    expect(all.find((r) => r.path === "btn")).toBeUndefined();
  });

  it("嵌套菜单子项路径正确拼接", () => {
    const all = routeConfigManager.getAllRoutes();
    expect(all.find((r) => r.path === "sub")).toBeDefined();
    expect(all.find((r) => r.path === "sub/child")).toBeDefined();
  });
});

describe("routeConfigManager path 解析", () => {
  beforeEach(seed);

  it("getRouteByPath 含/与不含 prefix 都命中", () => {
    expect(routeConfigManager.getRouteByPath("dashboard")?.path).toBe("dashboard");
    expect(routeConfigManager.getRouteByPath("/dashboard")?.path).toBe("dashboard");
  });

  it("getRouteMeta 返 meta", () => {
    expect(routeConfigManager.getRouteMeta("dashboard")?.title).toBe("仪表盘");
  });

  it("未知路径返 undefined", () => {
    expect(routeConfigManager.getRouteByPath("/no/such/path")).toBeUndefined();
    expect(routeConfigManager.getRouteMeta("/no/such/path")).toBeUndefined();
  });

  it("getRouteTitle 优先 meta.title, 否则从路径段提取", () => {
    expect(routeConfigManager.getRouteTitle("dashboard")).toBe("仪表盘");
    expect(routeConfigManager.getRouteTitle("/no/such/path")).toBe("path");
  });

  it("隐藏菜单仍注册(visible=0 仅影响菜单不显示,不删除路由)", () => {
    // 路由层面无 hidden 字段(默认 fallback 才用 meta.hidden = visible !== 1)
    // 当 caller 显式提供 meta 时,直接采用 caller 提供(meta 中无 hidden)
    const r = routeConfigManager.getRouteByPath("hidden");
    expect(r).toBeDefined();
    expect(r?.meta.title).toBe("隐藏");
    expect(r?.meta.hidden).toBeUndefined();
  });

  it("visible=0 + 无 caller meta 走 fallback 时 hidden=true", () => {
    routeConfigManager.clear();
    routeConfigManager.initialize([
      makeMenu({
        id: 9,
        menuName: "纯隐藏",
        path: "fallback-hidden",
        component: "pages/x/index",
        visible: 0,
        // meta 留空,触发 fallback 分支
      }),
    ] as Menu[]);
    const r = routeConfigManager.getRouteByPath("fallback-hidden");
    expect(r?.meta.hidden).toBe(true);
  });
});

describe("permission check", () => {
  beforeEach(seed);

  it("无 permissions 字段 → hasPermission=true", () => {
    expect(routeConfigManager.hasPermission("dashboard", [])).toEqual({ hasPermission: true });
  });

  it("permissions 命中返 true", () => {
    expect(
      routeConfigManager.hasPermission("user", ["system:user:list"])
    ).toEqual({ hasPermission: true });
  });

  it("permissions 部分缺失返 missingPermissions 数组", () => {
    const r = routeConfigManager.hasPermission("user", ["other:perm"]);
    expect(r.hasPermission).toBe(false);
    expect(r.missingPermissions).toEqual(["system:user:list"]);
  });

  it("permissions 多者任一命中即通过", () => {
    expect(
      routeConfigManager.hasPermission("user", ["a:1", "system:user:list"])
    ).toEqual({ hasPermission: true });
  });
});

describe("breadcrumb", () => {
  beforeEach(seed);

  it("已知路径生成全部 meta.title 段", () => {
    const bc = routeConfigManager.buildBreadcrumb("dashboard");
    expect(bc.length).toBe(1);
    expect(bc[0].title).toBe("仪表盘");
  });

  it("未知路径走 fallback(用 PATH_TRANSLATIONS)", () => {
    // PATH_TRANSLATIONS 仅有 "workstations"(复数) → "工位管理", "ops" → "运维管理"
    const bc = routeConfigManager.buildBreadcrumb("/ops/workstations");
    expect(bc.length).toBe(2);
    expect(bc[0].title).toBe("运维管理");
    expect(bc[1].title).toBe("工位管理");
  });

  it("未知段无翻译时 fallback 原文", () => {
    // "unknownSeg" 不在 PATH_TRANSLATIONS → 保留原文
    const bc = routeConfigManager.buildBreadcrumb("/unknownSeg/fooBar");
    expect(bc.length).toBe(2);
    expect(bc[0].title).toBe("unknownSeg");
    expect(bc[1].title).toBe("fooBar");
  });

  it("空路径返回首页", () => {
    const bc = routeConfigManager.buildBreadcrumb("/");
    expect(bc).toEqual([{ path: "/", title: "首页" }]);
  });
});
