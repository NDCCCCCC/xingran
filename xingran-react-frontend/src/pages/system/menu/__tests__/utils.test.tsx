/**
 * Phase 86 — menu utils 纯函数测试
 */
import { describe, it, expect } from "vitest";
import { flattenTree, buildParentOptions, calculateStatistics } from "../utils";

const menuTree: any[] = [
  {
    id: 1,
    menuName: "系统管理",
    menuType: "M",
    visible: 1,
    status: 0,
    children: [
      { id: 11, menuName: "用户管理", menuType: "C", visible: 1, status: 0 },
      { id: 12, menuName: "隐藏菜单", menuType: "C", visible: 0, status: 0 },
    ],
  },
  { id: 2, menuName: "监控", menuType: "M", visible: 1, status: 0 },
];

describe("flattenTree", () => {
  it("flattens menu tree DFS order", () => {
    const flat = flattenTree(menuTree);
    expect(flat.map((m) => m.id)).toEqual([1, 11, 12, 2]);
  });

  it("handles empty tree", () => {
    expect(flattenTree([])).toEqual([]);
  });
});

describe("buildParentOptions", () => {
  it("builds TreeSelect options from tree", () => {
    const opts = buildParentOptions(menuTree);
    expect(opts.length).toBeGreaterThan(0);
  });
});

describe("calculateStatistics", () => {
  it("computes menu statistics from flat list", () => {
    const flat = flattenTree(menuTree);
    const stats = calculateStatistics(flat);
    expect(stats).toBeDefined();
    expect(typeof stats.total).toBe("number");
  });
});
