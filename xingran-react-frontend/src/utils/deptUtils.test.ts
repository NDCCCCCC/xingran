import { describe, expect, it } from "vitest";
import {
  collectDescendantIds,
  dedupTreeByKey,
  filterExternalOrgDepts,
  findDeptNode,
  getDeptNodeId,
  toFullPathTree,
  toShortNameDataNode,
  trimTitleToLastSegment,
  type DeptLikeNode,
} from "./deptUtils";

describe("getDeptNodeId（多形状兼容）", () => {
  it("优先 id，缺省回落 value 再回落 key", () => {
    expect(getDeptNodeId({ id: "a", value: "b", key: "c" })).toBe("a");
    expect(getDeptNodeId({ value: "b", key: "c" })).toBe("b");
    expect(getDeptNodeId({ key: "c" })).toBe("c");
    expect(getDeptNodeId({})).toBe("");
  });
});

describe("filterExternalOrgDepts（外部机构过滤）", () => {
  it("仅保留 externalOrg 节点及其祖先链，纯内部子树被丢弃", () => {
    const nodes: DeptLikeNode[] = [
      { id: "A", isExternalOrg: 1, children: [{ id: "A1" }] },
      { id: "B", isExternalOrg: 0, children: [{ id: "B1", isExternalOrg: 1 }] },
      { id: "C" },
    ];
    const result = filterExternalOrgDepts(nodes);
    expect(result.map((n) => n.id)).toEqual(["A", "B"]);
    // A 自身是 externalOrg 被保留，但其非 externalOrg 子节点 A1 无后代可保留 → 被丢弃
    expect(result[0].children).toBeUndefined();
    // B 仅作为 B1 的祖先链保留，children 只剩 externalOrg 子树
    expect(result[1].children?.map((c) => c.id)).toEqual(["B1"]);
  });

  it("无 externalOrg 后代的节点不保留空 children 壳", () => {
    const result = filterExternalOrgDepts([{ id: "X", isExternalOrg: 0, children: [] }]);
    expect(result).toEqual([]);
  });

  it("null 输入安全返回空数组", () => {
    expect(filterExternalOrgDepts(null as unknown as DeptLikeNode[])).toEqual([]);
  });
});

describe("findDeptNode（深度优先查找）", () => {
  const tree: DeptLikeNode[] = [
    { id: "1", children: [{ id: "1-1", children: [{ id: "1-1-1" }] }] },
    { id: "2" },
  ];

  it("命中根节点", () => {
    expect(findDeptNode(tree, "2")?.id).toBe("2");
  });

  it("命中深层节点（返回原引用）", () => {
    const deep = tree[0].children?.[0].children?.[0];
    expect(findDeptNode(tree, "1-1-1")).toBe(deep);
  });

  it("未找到返回 null", () => {
    expect(findDeptNode(tree, "404")).toBeNull();
  });
});

describe("collectDescendantIds（收集子树）", () => {
  const tree: DeptLikeNode[] = [
    {
      id: "root",
      children: [{ id: "c1", children: [{ id: "g1" }] }, { id: "c2" }],
    },
    { id: "other" },
  ];

  it("收集自身 + 全部后代", () => {
    expect(collectDescendantIds(tree, "root")).toEqual(["root", "c1", "g1", "c2"]);
    expect(collectDescendantIds(tree, "c1")).toEqual(["c1", "g1"]);
  });

  it("叶子节点只返回自身", () => {
    expect(collectDescendantIds(tree, "g1")).toEqual(["g1"]);
  });

  it("不存在的 id 返回空数组", () => {
    expect(collectDescendantIds(tree, "404")).toEqual([]);
  });
});

describe("trimTitleToLastSegment（title 收窄）", () => {
  it("去掉祖先路径只留最后一段，递归处理 children", () => {
    const nodes = [
      {
        title: "A / B / C",
        children: [{ title: "A / B / C / D" }, { title: "无路径" }],
      },
    ];
    const result = trimTitleToLastSegment(nodes);
    expect(result[0].title).toBe("C");
    expect(result[0].children?.[0].title).toBe("D");
    expect(result[0].children?.[1].title).toBe("无路径");
  });
});

describe("toFullPathTree（全路径转换）", () => {
  const tree = [
    {
      id: "1",
      deptName: "集团",
      children: [{ id: "2", deptName: "分公司", children: [{ id: "3", deptName: "人力资源部" }] }],
    },
  ];

  it("默认拼接祖先全路径并透传 isExternalOrg", () => {
    const result = toFullPathTree([{ ...tree[0], isExternalOrg: 1 }]);
    expect(result[0]).toMatchObject({
      title: "集团",
      value: "1",
      key: "1",
      isExternalOrg: 1,
    });
    expect(result[0].children?.[0]).toMatchObject({ title: "集团 / 分公司", value: "2" });
    expect(result[0].children?.[0].children?.[0].title).toBe("集团 / 分公司 / 人力资源部");
  });

  it("startFromLevel=2 丢弃顶级祖先（旧 convertDeptTreeData 语义）", () => {
    const result = toFullPathTree(tree, { startFromLevel: 2 });
    expect(result[0].title).toBe("集团"); // 顶级不受影响
    expect(result[0].children?.[0].title).toBe("分公司"); // ancestors[0] 被丢弃
    expect(result[0].children?.[0].children?.[0].title).toBe("分公司 / 人力资源部");
  });

  it("缺 deptName 的节点 title 为空字符串", () => {
    const result = toFullPathTree([{ id: "x" }]);
    expect(result[0].title).toBe("");
  });
});

describe("toShortNameDataNode（短名转换）", () => {
  it("title 只显示短名，叶子节点标记 isLeaf", () => {
    const result = toShortNameDataNode([
      { id: "1", deptName: "集团", children: [{ id: "2", deptName: "分公司" }] },
    ]);
    expect(result[0]).toMatchObject({ title: "集团", key: "1", value: "1", isLeaf: false });
    expect(result[0].children?.[0]).toMatchObject({ title: "分公司", isLeaf: true });
    expect(result[0].children?.[0].children).toBeUndefined();
  });
});

describe("dedupTreeByKey（同 key 去重）", () => {
  it("深度优先保留首次出现（后端重复提升数据的兜底场景）", () => {
    // "2" 同时作为 "1" 的子节点与顶层重复出现（后端 buildDeptTree 历史数据形态）
    const nodes = [{ value: "1", children: [{ value: "2" }] }, { value: "2" }, { value: "3" }];
    const result = dedupTreeByKey(nodes);
    expect(result.map((n) => n.value)).toEqual(["1", "3"]);
    expect(result[0].children?.map((c) => c.value)).toEqual(["2"]);
  });

  it("空 id 节点被丢弃", () => {
    expect(dedupTreeByKey([{ value: undefined, key: undefined }])).toEqual([]);
  });
});
