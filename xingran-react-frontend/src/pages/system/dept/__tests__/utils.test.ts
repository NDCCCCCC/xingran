/**
 * Phase 86 — dept utils 纯函数测试
 */
import { describe, it, expect } from "vitest";
import { flattenTreeToList, transformToParentTreeOptions, renderTreeData } from "../utils";

const tree: any[] = [
  {
    id: "d1",
    deptName: "总经办",
    children: [
      { id: "d2", deptName: "技术部", children: [{ id: "d3", deptName: "前端组" }] },
      { id: "d4", deptName: "综合部" },
    ],
  },
  { id: "d5", deptName: "财务部" },
];

describe("flattenTreeToList", () => {
  it("flattens nested tree into flat list (DFS)", () => {
    const list = flattenTreeToList(tree);
    expect(list).toHaveLength(5);
    expect(list.map((d) => d.id)).toEqual(["d1", "d2", "d3", "d4", "d5"]);
  });

  it("returns empty for null/invalid input", () => {
    expect(flattenTreeToList(null as any)).toEqual([]);
    expect(flattenTreeToList(undefined as any)).toEqual([]);
    expect(flattenTreeToList("bad" as any)).toEqual([]);
  });

  it("returns flat array as-is for leaf-only tree", () => {
    const leaves: any[] = [{ id: "a" }, { id: "b" }];
    expect(flattenTreeToList(leaves)).toHaveLength(2);
  });
});

describe("transformToParentTreeOptions", () => {
  it("maps tree to TreeSelect options with title/value/key", () => {
    const opts = transformToParentTreeOptions(tree);
    expect(opts[0].title).toBe("总经办");
    expect(opts[0].value).toBe("d1");
    expect(opts[0].children).toHaveLength(2);
  });

  it("leaf children become undefined", () => {
    const opts = transformToParentTreeOptions(tree);
    const leaf = opts[0].children![1];
    expect(leaf.children).toBeUndefined();
  });

  it("returns empty array for invalid input", () => {
    expect(transformToParentTreeOptions(null as any)).toEqual([]);
  });
});

describe("renderTreeData", () => {
  it("adds key field to every node", () => {
    const rows = renderTreeData(tree);
    expect(rows[0].key).toBe("d1");
  });

  it("returns empty array for invalid input", () => {
    expect(renderTreeData(null as any)).toEqual([]);
  });
});
