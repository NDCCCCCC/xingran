/**
 * Phase 88 Batch339 — pages/system/user/utils 测试
 */
import { describe, it, expect } from "vitest";
import { renderToString } from "react-dom/server";
import { renderDeptTreeOptions, formatGender, formatStatus } from "../utils";

describe("pages/system/user/utils", () => {
  describe("renderDeptTreeOptions", () => {
    it("空 → 暂无部门数据", () => {
      const result = renderDeptTreeOptions([]);
      expect(result.length).toBe(1);
    });

    it("null → 暂无部门数据", () => {
      const result = renderDeptTreeOptions(null as any);
      expect(result.length).toBe(1);
    });

    it("undefined → 暂无部门数据", () => {
      const result = renderDeptTreeOptions(undefined as any);
      expect(result.length).toBe(1);
    });

    it("扁平部门 → 1 个 Option", () => {
      const result = renderDeptTreeOptions([{ id: "d1", deptName: "Dept 1" }]);
      expect(result.length).toBe(1);
    });

    it("嵌套部门 → 多个 Options + 缩进", () => {
      const result = renderDeptTreeOptions([
        {
          id: "d1",
          deptName: "Root",
          children: [{ id: "d2", deptName: "Child" }],
        },
      ]);
      expect(result.length).toBe(2);
    });

    it("3 层嵌套 → 3 Options", () => {
      const result = renderDeptTreeOptions([
        {
          id: "d1",
          deptName: "L1",
          children: [
            {
              id: "d2",
              deptName: "L2",
              children: [{ id: "d3", deptName: "L3" }],
            },
          ],
        },
      ]);
      expect(result.length).toBe(3);
    });

    it("空 children 不渲染子 Option", () => {
      const result = renderDeptTreeOptions([{ id: "d1", deptName: "Root", children: [] }]);
      expect(result.length).toBe(1);
    });
  });

  describe("formatGender", () => {
    it("0 → 男 (fallback)", () => {
      expect(formatGender(0)).toBe("男");
    });

    it("1 → 女 (fallback)", () => {
      expect(formatGender(1)).toBe("女");
    });

    it("未知 → 保密", () => {
      expect(formatGender(99)).toBe("保密");
    });

    it("dict 命中 dictLabel", () => {
      expect(formatGender(0, [{ dictValue: "0", dictLabel: "Male" }])).toBe("Male");
    });

    it("dict 未命中 → fallback", () => {
      expect(formatGender(0, [{ dictValue: "5", dictLabel: "Other" }])).toBe("男");
    });

    it("dict 空 → fallback", () => {
      expect(formatGender(1, [])).toBe("女");
    });

    it("dict 空 + 未知 → 保密", () => {
      expect(formatGender(99, [])).toBe("保密");
    });
  });

  describe("formatStatus", () => {
    it("0 → 启用", () => {
      expect(formatStatus(0)).toEqual({ text: "启用", color: "success" });
    });

    it("1 → 禁用", () => {
      expect(formatStatus(1)).toEqual({ text: "禁用", color: "error" });
    });

    it("未知 → 默认", () => {
      expect(formatStatus(99)).toEqual({ text: "未知", color: "default" });
    });
  });
});
