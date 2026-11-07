/**
 * Phase 133 / DS-06：DeptSidebar ↔ 后端可见集联动断言。
 *
 * 核心验收：**两个不同 data_scope 的用户渲染出不同的部门集**。
 * 可见集（visibleDeptIds）来自后端 tree/dept-options 已收口的结果，前端不硬编码
 * 任何过滤规则；本测试用两个不同的后端可见集模拟两个 scope 用户。
 *
 * 覆盖：
 *   - DeptTree 组件集成：不同 visibleDeptIds → 渲染部门集不同；
 *   - 未传 visibleDeptIds → 完整渲染后端 tree（零回归）；
 *   - 纯函数 filterTreeByVisibleDeptIds 的树形保留 / 祖先路径 / 空集 / 空入参分支。
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

// 后端 /system/departments/tree 返回的「收口前」完整树（由后端按 data_scope 过滤后
// 把可见 ID 集通过 dept-options 返回前端）。这里模拟后端返回的可见集不同 = 用户不同。
const MOCK_DEPTS = [
  {
    id: "d-root",
    deptName: "总部",
    parentId: "0",
    children: [
      {
        id: "d-dev",
        deptName: "研发部",
        parentId: "d-root",
        children: [
          { id: "d-fe", deptName: "前端组", parentId: "d-dev" },
          { id: "d-be", deptName: "后端组", parentId: "d-dev" },
        ],
      },
      {
        id: "d-ops",
        deptName: "运维部",
        parentId: "d-root",
        children: [{ id: "d-net", deptName: "网络组", parentId: "d-ops" }],
      },
    ],
  },
];

vi.mock("@/hooks/useDeptTree", () => ({
  useDeptTree: () => ({ data: MOCK_DEPTS, isLoading: false }),
}));

import DeptTree from "../index";
import { filterTreeByVisibleDeptIds } from "../visibleDeptFilter";

describe("Phase 133 / DeptSidebar 可见集联动", () => {
  it("两个不同 scope 用户（不同 visibleDeptIds）渲染不同部门集", () => {
    // 用户 A（如 Dept 档）：仅可见「研发部」子树
    const { unmount: unmountA } = render(<DeptTree visibleDeptIds={["d-dev", "d-fe", "d-be"]} />);
    expect(screen.getByText("研发部")).toBeInTheDocument();
    expect(screen.queryByText("运维部")).not.toBeInTheDocument();
    expect(screen.queryByText("网络组")).not.toBeInTheDocument();
    unmountA();

    // 用户 B（如另一部门档）：仅可见「运维部」子树
    const { unmount: unmountB } = render(<DeptTree visibleDeptIds={["d-ops", "d-net"]} />);
    expect(screen.getByText("运维部")).toBeInTheDocument();
    expect(screen.queryByText("研发部")).not.toBeInTheDocument();
    expect(screen.queryByText("前端组")).not.toBeInTheDocument();
    unmountB();

    // 用户 C（All 档）：后端返回全量可见集 → 两个顶层部门都渲染。
    // 注意：antd Tree 默认折叠子树，子节点 DOM 不一定挂载，故这里只断言顶层节点；
    // 「全量子树保留」由下方纯函数测试（自身命中 → 连同全部后代保留）覆盖。
    const { unmount: unmountC } = render(
      <DeptTree visibleDeptIds={["d-root", "d-dev", "d-fe", "d-be", "d-ops", "d-net"]} />
    );
    expect(screen.getByText("研发部")).toBeInTheDocument();
    expect(screen.getByText("运维部")).toBeInTheDocument();
    unmountC();
  });

  it("未传 visibleDeptIds → 以后端 tree 结果为准，完整渲染（零回归）", () => {
    render(<DeptTree />);
    expect(screen.getByText("总部")).toBeInTheDocument();
    expect(screen.getByText("研发部")).toBeInTheDocument();
    expect(screen.getByText("运维部")).toBeInTheDocument();
  });

  it("empty visibleDeptIds → 以后端结果为准，不做前端额外裁剪", () => {
    render(<DeptTree visibleDeptIds={[]} />);
    expect(screen.getByText("总部")).toBeInTheDocument();
  });

  it("fail-closed 可见集（空集语义由后端给出）→ 不渲染越权部门", () => {
    // 后端异常枚举档返回空可见集；此处用「不存在的 ID」模拟集合过滤后的空交集。
    render(<DeptTree visibleDeptIds={["d-nonexistent"]} />);
    expect(screen.queryByText("研发部")).not.toBeInTheDocument();
    expect(screen.queryByText("运维部")).not.toBeInTheDocument();
  });
});

describe("filterTreeByVisibleDeptIds 纯函数", () => {
  const tree = [
    { key: "a", children: [{ key: "a1" }, { key: "a2" }] },
    { key: "b", children: [{ key: "b1" }] },
    { key: "c" },
  ];

  it("visibleDeptIds 为空/undefined → 原样返回同一引用", () => {
    expect(filterTreeByVisibleDeptIds(tree)).toBe(tree);
    expect(filterTreeByVisibleDeptIds(tree, [])).toBe(tree);
  });

  it("祖先未命中但子树命中 → 保留祖先作为路径，子树裁剪", () => {
    const out = filterTreeByVisibleDeptIds(tree, ["a1"]);
    expect(out).toHaveLength(1);
    expect(out[0].key).toBe("a");
    expect(out[0].children?.map((c) => c.key)).toEqual(["a1"]);
  });

  it("自身命中 → 连同全部后代保留", () => {
    const out = filterTreeByVisibleDeptIds(tree, ["a"]);
    expect(out).toHaveLength(1);
    expect(out[0].children?.map((c) => c.key)).toEqual(["a1", "a2"]);
  });

  it("全不命中 → 空集（fail-closed）", () => {
    expect(filterTreeByVisibleDeptIds(tree, ["zzz"])).toEqual([]);
  });

  it("多个 scope 可见集产生不同结果（联动不变量）", () => {
    const devOnly = filterTreeByVisibleDeptIds(tree, ["a"]);
    const opsOnly = filterTreeByVisibleDeptIds(tree, ["b"]);
    expect(devOnly[0].key).toBe("a");
    expect(opsOnly[0].key).toBe("b");
    expect(devOnly).not.toEqual(opsOnly);
  });
});
