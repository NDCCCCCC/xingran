/**
 * Phase 88 Batch203 — components/layout/sidebar-helper 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import {
  buildMenuPathMap,
  isSameTopLevelMenu,
  getMenuLevel,
  isThirdLevelMenu,
  isSecondLevelMenu,
  isTopLevelMenu,
} from "../sidebar-helper";

const sampleMenus: any[] = [
  {
    id: "m1",
    menuType: "C",
    visible: 1,
    path: "/system",
    children: [
      {
        id: "m11",
        menuType: "C",
        visible: 1,
        path: "/system/user",
        children: [{ id: "m111", menuType: "F", visible: 1, path: "/system/user/list" }],
      },
    ],
  },
  { id: "m2", menuType: "C", visible: 1, path: "/dashboard" },
];

describe("layout/sidebar-helper", () => {
  it("buildMenuPathMap 顶层", () => {
    const map = buildMenuPathMap(sampleMenus);
    expect(map.has("/system")).toBe(true);
    expect(map.has("/dashboard")).toBe(true);
  });

  it("buildMenuPathMap 二级", () => {
    const map = buildMenuPathMap(sampleMenus);
    const info = map.get("/system/user");
    expect(info?.level).toBe(2);
    expect(info?.topLevel).toBe("m1");
  });

  it("buildMenuPathMap 三级", () => {
    // 三级菜单不能用 F 类型(menuType=F 按钮),要 C 类
    const menus = [
      {
        id: "m1",
        menuType: "C",
        visible: 1,
        path: "/system",
        children: [
          {
            id: "m11",
            menuType: "C",
            visible: 1,
            path: "/system/user",
            children: [{ id: "m111", menuType: "C", visible: 1, path: "/system/user/list" }],
          },
        ],
      },
    ];
    const map = buildMenuPathMap(menus);
    const info = map.get("/system/user/list");
    expect(info?.level).toBe(3);
    expect(info?.secondLevel).toBe("m11");
  });

  it("buildMenuPathMap 跳过 F 类型", () => {
    const map = buildMenuPathMap(sampleMenus);
    expect(map.get("m111")).toBeUndefined();
  });

  it("buildMenuPathMap 跳过不可见", () => {
    const menus = [{ id: "hidden", menuType: "C", visible: 0, path: "/hidden" }];
    const map = buildMenuPathMap(menus);
    expect(map.has("/hidden")).toBe(false);
  });

  it("buildMenuPathMap 存 ID key", () => {
    const map = buildMenuPathMap(sampleMenus);
    expect(map.has("m2")).toBe(true);
  });

  it("isSameTopLevelMenu 同组", () => {
    const map = buildMenuPathMap(sampleMenus);
    expect(isSameTopLevelMenu("/system", "/system/user", map)).toBe(true);
  });

  it("isSameTopLevelMenu 不同组", () => {
    const map = buildMenuPathMap(sampleMenus);
    expect(isSameTopLevelMenu("/system", "/dashboard", map)).toBe(false);
  });

  it("isSameTopLevelMenu 缺 key", () => {
    const map = buildMenuPathMap(sampleMenus);
    expect(isSameTopLevelMenu("/unknown", "/system", map)).toBe(false);
  });

  it("getMenuLevel 返回层级", () => {
    const map = buildMenuPathMap(sampleMenus);
    expect(getMenuLevel("/system", map)).toBe(1);
    expect(getMenuLevel("/system/user", map)).toBe(2);
    expect(getMenuLevel("/unknown", map)).toBe(0);
  });

  it("isThirdLevelMenu 三级 true", () => {
    const menus = [
      {
        id: "m1",
        menuType: "C",
        visible: 1,
        path: "/system",
        children: [
          {
            id: "m11",
            menuType: "C",
            visible: 1,
            path: "/system/user",
            children: [{ id: "m111", menuType: "C", visible: 1, path: "/system/user/list" }],
          },
        ],
      },
    ];
    const map = buildMenuPathMap(menus);
    expect(isThirdLevelMenu("/system/user/list", map)).toBe(true);
  });

  it("isSecondLevelMenu 二级 true", () => {
    const map = buildMenuPathMap(sampleMenus);
    expect(isSecondLevelMenu("/system/user", map)).toBe(true);
  });

  it("isTopLevelMenu 一级 true", () => {
    const map = buildMenuPathMap(sampleMenus);
    expect(isTopLevelMenu("/system", map)).toBe(true);
  });
});
