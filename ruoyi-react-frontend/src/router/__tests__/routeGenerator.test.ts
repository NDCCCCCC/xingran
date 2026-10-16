/**
 * Phase 88 Batch17c — routeGenerator 安全验证单测(路径遍历/XSS/组件解析)
 */
import { describe, it, expect, vi } from "vitest";
import { RouteGenerator } from "../routeGenerator";
import type { Menu } from "@/types";

const mkMenu = (over: Partial<Menu> = {}): Menu =>
  ({
    id: "m1",
    menuName: "系统管理",
    path: "system",
    menuType: "C",
    visible: 1,
    children: [],
    ...over,
  }) as unknown as Menu;

describe("RouteGenerator.generate", () => {
  it("returns empty for empty/null menus", () => {
    expect(RouteGenerator.generate([])).toEqual([]);
    expect(RouteGenerator.generate(null as unknown as Menu[])).toEqual([]);
  });

  it("builds route for C-type menu", () => {
    const routes = RouteGenerator.generate([mkMenu()]);
    expect(routes).toHaveLength(1);
    expect(routes[0].path).toBe("system");
    expect(routes[0].component).toBe("pages/system/index");
  });

  it("concatenates child paths under parent", () => {
    const menus = [
      mkMenu({
        children: [mkMenu({ id: "m2", menuName: "用户", path: "user" })],
      }),
    ];
    const routes = RouteGenerator.generate(menus);
    expect(routes).toHaveLength(2);
    expect(routes[1].path).toBe("system/user");
  });

  it("rejects path traversal (../)", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const routes = RouteGenerator.generate([mkMenu({ path: "../etc/passwd" })]);
    expect(routes).toHaveLength(0);
    warn.mockRestore();
  });

  it("rejects backslash paths", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const routes = RouteGenerator.generate([mkMenu({ path: "a\\b" })]);
    expect(routes).toHaveLength(0);
    warn.mockRestore();
  });

  it("rejects .html/.js paths", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    expect(RouteGenerator.generate([mkMenu({ path: "evil.html" })])).toHaveLength(0);
    expect(RouteGenerator.generate([mkMenu({ path: "evil.js" })])).toHaveLength(0);
    warn.mockRestore();
  });

  it("rejects missing id/menuName", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    expect(RouteGenerator.generate([mkMenu({ id: "" })])).toHaveLength(0);
    expect(RouteGenerator.generate([mkMenu({ menuName: "" })])).toHaveLength(0);
    warn.mockRestore();
  });

  it("rejects XSS in meta.title (script/javascript:/onerror=)", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    for (const title of [
      "<script>alert(1)</script>",
      "javascript:alert(1)",
      'x" onerror="alert(1)',
      'x" onload="alert(1)',
    ]) {
      const routes = RouteGenerator.generate([
        mkMenu({ meta: { title } as unknown as Menu["meta"] }),
      ]);
      expect(routes).toHaveLength(0);
    }
    warn.mockRestore();
  });
});

describe("RouteGenerator.resolveComponent (via generate)", () => {
  it("dashboard path maps to dashboard-system", () => {
    const routes = RouteGenerator.generate([mkMenu({ path: "dashboard" })]);
    expect(routes[0].component).toBe("pages/dashboard-system/index");
  });

  it("explicit component preserved with pages/ prefix normalization", () => {
    const routes = RouteGenerator.generate([
      mkMenu({ path: "x", component: "monitor/server/index" }),
    ]);
    expect(routes[0].component).toBe("pages/monitor/server/index");
  });

  it("leading slash stripped", () => {
    const routes = RouteGenerator.generate([mkMenu({ path: "x", component: "/monitor/logs" })]);
    expect(routes[0].component).toBe("pages/monitor/logs/index");
  });

  it("absolute path (starts with /) wins over parent concat", () => {
    const menus = [
      mkMenu({
        children: [mkMenu({ id: "m2", menuName: "独立", path: "/standalone" })],
      }),
    ];
    const routes = RouteGenerator.generate(menus);
    expect(routes[1].path).toBe("standalone");
  });

  it("menuType M/D without C children yields no routes", () => {
    const routes = RouteGenerator.generate([mkMenu({ menuType: "M" })]);
    expect(routes).toHaveLength(0);
  });

  it("meta fallback built from menu fields (visible/affix)", () => {
    const routes = RouteGenerator.generate([mkMenu({ meta: undefined })]);
    expect(routes[0].meta.title).toBe("系统管理");
    expect(routes[0].meta.hidden).toBe(false);
    expect(routes[0].meta.affix).toBe(false);
  });

  it("getComponentPattern returns glob", () => {
    expect(RouteGenerator.getComponentPattern()).toBe("/src/pages/**/{index,detail}.tsx");
  });
});
