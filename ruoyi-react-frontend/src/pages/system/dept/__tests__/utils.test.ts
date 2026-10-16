/**
 * Phase 88 Batch356 — pages/system/dept/utils 测试
 */
import { describe, it, expect } from "vitest";
import { flattenTreeToList, transformToParentTreeOptions, renderTreeData } from "../utils";

describe("pages/system/dept/utils", () => {
  describe("flattenTreeToList", () => {
    it("扁平化嵌套树", () => {
      const tree: any[] = [
        {
          id: "d1",
          deptName: "Root",
          children: [
            { id: "d2", deptName: "Child 1" },
            { id: "d3", deptName: "Child 2" },
          ],
        },
      ];
      const result = flattenTreeToList(tree);
      expect(result.length).toBe(3);
      expect(result[0].id).toBe("d1");
      expect(result[2].id).toBe("d3");
    });

    it("空 → []", () => {
      expect(flattenTreeToList([])).toEqual([]);
    });

    it("null → []", () => {
      expect(flattenTreeToList(null as any)).toEqual([]);
    });

    it("undefined → []", () => {
      expect(flattenTreeToList(undefined as any)).toEqual([]);
    });

    it("非数组 → []", () => {
      expect(flattenTreeToList("x" as any)).toEqual([]);
    });

    it("深层递归", () => {
      const tree: any[] = [
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
      ];
      expect(flattenTreeToList(tree).length).toBe(3);
    });
  });

  describe("transformToParentTreeOptions", () => {
    it("转 title/value/key", () => {
      const tree: any[] = [{ id: "d1", deptName: "Root", children: [] }];
      const result = transformToParentTreeOptions(tree);
      expect(result[0]).toEqual({ title: "Root", value: "d1", key: "d1", children: undefined });
    });

    it("递归 children", () => {
      const tree: any[] = [
        {
          id: "d1",
          deptName: "Root",
          children: [{ id: "d2", deptName: "Child" }],
        },
      ];
      const result = transformToParentTreeOptions(tree);
      expect(result[0].children?.length).toBe(1);
      expect(result[0].children?.[0].title).toBe("Child");
    });

    it("空 → []", () => {
      expect(transformToParentTreeOptions([])).toEqual([]);
    });

    it("null → []", () => {
      expect(transformToParentTreeOptions(null as any)).toEqual([]);
    });

    it("无 children → children undefined", () => {
      const result = transformToParentTreeOptions([{ id: "x", deptName: "X" } as any]);
      expect(result[0].children).toBeUndefined();
    });
  });

  describe("renderTreeData", () => {
    it("递归 + key", () => {
      const tree: any[] = [
        { id: "d1", deptName: "Root", children: [{ id: "d2", deptName: "Child" }] },
      ];
      const result = renderTreeData(tree);
      expect(result[0].key).toBe("d1");
      expect(result[0].children?.[0].key).toBe("d2");
    });

    it("空 → []", () => {
      expect(renderTreeData([])).toEqual([]);
    });

    it("null → []", () => {
      expect(renderTreeData(null as any)).toEqual([]);
    });
  });
});
