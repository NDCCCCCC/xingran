/**
 * 回归：hover 预取必须走「路由路径 → 菜单 component → 真实组件路径」的映射，
 * 不能把路由 URL 直接当组件路径。
 *
 * 为什么必须用菜单数据映射（字符串推导会失败）：
 *   路由 /assets/dashboard    → 组件 pages/asset/reconciliation/dashboard/index.tsx
 *   路由 /dashboard           → 组件 pages/dashboard-system/index.tsx（渲染特例）
 *   路由 /network/discoveries → 组件 pages/network/discoveries/index.tsx
 *
 * 注：本文件只断言「映射结果」与「映射目标确实存在于 glob」，
 * 不触发真实 import —— 真实 chunk 在 vitest 下 transform 极慢（>30s），
 * 属于测试环境代价问题，与被测逻辑无关。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  ComponentLoader,
  componentLoader,
  preloadRouteComponent,
  resolveRouteComponentPath,
} from "../componentLoader";
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

/** glob 中是否存在该组件（= 预取能否命中真实 chunk） */
const globHas = (componentPath: string): boolean =>
  Boolean(ComponentLoader.componentModules[`/src/${componentPath}`]);

describe("路由路径 → 组件路径映射", () => {
  beforeEach(() => {
    routeConfigManager.clear();
    componentLoader.clearCache();
  });

  afterEach(() => {
    routeConfigManager.clear();
    componentLoader.clearCache();
  });

  it("菜单 component 与路由段不一致时仍映射到真实 chunk", () => {
    routeConfigManager.initialize([
      makeMenu({
        id: 1,
        menuName: "资产管理",
        path: "assets",
        menuType: "M",
        component: "Layout",
        children: [
          makeMenu({
            id: 2,
            parentId: 1,
            menuName: "对账仪表盘",
            path: "dashboard",
            menuType: "C",
            component: "asset/reconciliation/dashboard/index",
          }),
        ],
      }),
    ]);

    const resolved = resolveRouteComponentPath("/assets/dashboard");

    expect(resolved).toBe("pages/asset/reconciliation/dashboard/index.tsx");
    expect(globHas(resolved!)).toBe(true);
  });

  it("普通路由：/network/discoveries → pages/network/discoveries/index.tsx", () => {
    routeConfigManager.initialize([
      makeMenu({
        id: 3,
        menuName: "设备发现",
        path: "network/discoveries",
        menuType: "C",
        component: "network/discoveries/index",
      }),
    ]);

    const resolved = resolveRouteComponentPath("/network/discoveries");

    expect(resolved).toBe("pages/network/discoveries/index.tsx");
    expect(globHas(resolved!)).toBe(true);
  });

  it("dashboard 路由复用渲染特例（pages/dashboard-system/index.tsx）", () => {
    routeConfigManager.initialize([
      makeMenu({
        id: 4,
        menuName: "仪表盘",
        path: "dashboard",
        menuType: "C",
        component: "dashboard/index",
      }),
    ]);

    const resolved = resolveRouteComponentPath("/dashboard");

    expect(resolved).toBe("pages/dashboard-system/index.tsx");
    expect(globHas(resolved!)).toBe(true);
  });

  it("未知路由解析为 null", () => {
    routeConfigManager.initialize([
      makeMenu({
        id: 5,
        menuName: "用户管理",
        path: "system/user",
        menuType: "C",
        component: "system/user/index",
      }),
    ]);

    expect(resolveRouteComponentPath("/not/exists")).toBeNull();
  });

  it("路由配置未初始化时解析为 null", () => {
    expect(resolveRouteComponentPath("/network/discoveries")).toBeNull();
  });
});

describe("preloadRouteComponent：预取失败静默", () => {
  let errorSpy: ReturnType<typeof vi.spyOn>;
  let warnSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    routeConfigManager.clear();
    componentLoader.clearCache();
    errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
    routeConfigManager.clear();
    componentLoader.clearCache();
  });

  it("未知路由静默跳过，不污染控制台", async () => {
    routeConfigManager.initialize([
      makeMenu({
        id: 5,
        menuName: "用户管理",
        path: "system/user",
        menuType: "C",
        component: "system/user/index",
      }),
    ]);

    await expect(preloadRouteComponent("/not/exists")).resolves.toBe(false);

    expect(errorSpy).not.toHaveBeenCalled();
    expect(warnSpy).not.toHaveBeenCalled();
  });

  it("菜单 component 指向不存在的组件时静默失败", async () => {
    routeConfigManager.initialize([
      makeMenu({
        id: 6,
        menuName: "幽灵菜单",
        path: "ghost",
        menuType: "C",
        component: "ghost/not-exist/index",
      }),
    ]);

    await expect(preloadRouteComponent("/ghost")).resolves.toBe(false);

    expect(errorSpy).not.toHaveBeenCalled();
    expect(warnSpy).not.toHaveBeenCalled();
  });
});
