/**
 * Phase 88 Batch338 — pages/system/menu/utils 测试
 */
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import {
  flattenTree,
  buildParentOptions,
  renderTreeData,
  calculateStatistics,
  renderMenuName,
} from "../utils";

const sampleMenus: any[] = [
  {
    id: "m1",
    menuName: "系统",
    menuType: "M",
    children: [
      { id: "m2", menuName: "用户", menuType: "C" },
      { id: "m3", menuName: "角色", menuType: "C" },
    ],
  },
  { id: "m4", menuName: "Button", menuType: "F" },
];

describe("pages/system/menu/utils", () => {
  describe("flattenTree", () => {
    it("扁平化嵌套树", () => {
      const result = flattenTree(sampleMenus);
      expect(result.length).toBe(4);
      expect(result[0].id).toBe("m1");
      expect(result[3].id).toBe("m4");
    });

    it("空数组 → 空", () => {
      expect(flattenTree([])).toEqual([]);
    });

    it("null → 空", () => {
      expect(flattenTree(null as any)).toEqual([]);
    });

    it("undefined → 空", () => {
      expect(flattenTree(undefined as any)).toEqual([]);
    });

    it("非数组 → 空", () => {
      expect(flattenTree("not array" as any)).toEqual([]);
    });

    it("无 children 的项保留", () => {
      expect(flattenTree([{ id: "x", menuName: "X", menuType: "C" }]).length).toBe(1);
    });
  });

  describe("buildParentOptions", () => {
    it("包含顶级菜单", () => {
      const result = buildParentOptions([]);
      expect(result[0]).toEqual({ title: "顶级菜单", value: "", key: "" });
    });

    it("递归累积前缀", () => {
      const result = buildParentOptions(sampleMenus);
      expect(result.length).toBe(5); // 顶级 + 4 项
      expect(result.find((o) => o.value === "m1")?.title).toBe("系统");
      expect(result.find((o) => o.value === "m2")?.title).toBe("系统 / 用户");
    });

    it("null → 仅顶级", () => {
      const result = buildParentOptions(null as any);
      expect(result.length).toBe(1);
    });
  });

  describe("renderTreeData", () => {
    it("递归 + key 字段", () => {
      const result = renderTreeData(sampleMenus);
      expect(result[0].key).toBe("m1");
      expect(result[0].children?.length).toBe(2);
    });

    it("无 children 不渲染字段", () => {
      const result = renderTreeData([{ id: "x", menuName: "X" }]);
      expect(result[0].children).toBeUndefined();
    });
  });

  describe("calculateStatistics", () => {
    it("统计 total/directories/menus/buttons", () => {
      const flat = flattenTree(sampleMenus);
      const stats = calculateStatistics(flat);
      expect(stats.total).toBe(4);
      expect(stats.directories).toBe(1);
      expect(stats.menus).toBe(2);
      expect(stats.buttons).toBe(1);
    });

    it("空 → 全 0", () => {
      const stats = calculateStatistics([]);
      expect(stats.total).toBe(0);
      expect(stats.directories).toBe(0);
    });
  });

  describe("renderMenuName", () => {
    it("有 icon 时渲染 icon span", () => {
      const { container } = render(
        <>{renderMenuName({ id: "x", menuName: "X", menuType: "C", icon: "user" } as any)}</>
      );
      expect(container.textContent).toContain("X");
    });

    it("无 icon 走 menuType default", () => {
      const { container } = render(
        <>{renderMenuName({ id: "x", menuName: "Y", menuType: "C" } as any)}</>
      );
      expect(container.textContent).toContain("Y");
    });

    it("无 icon + menuType M", () => {
      const { container } = render(
        <>{renderMenuName({ id: "x", menuName: "Z", menuType: "M" } as any)}</>
      );
      expect(container.textContent).toContain("Z");
    });

    it("menuName 文本始终渲染", () => {
      render(<>{renderMenuName({ id: "x", menuName: "TestName", menuType: "C" } as any)}</>);
      expect(screen.getByText("TestName")).toBeInTheDocument();
    });
  });
});
