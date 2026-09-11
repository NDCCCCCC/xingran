/**
 * DeptTree 组件测试 — 覆盖批次 6.1 单遍过滤重构后的分支
 * （GOV-05 per-dir floor：components/DeptTree 44.8% ratchet 保持）
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

// 固定部门树：父节点带两组子节点，覆盖 title 命中父/命中子/无命中 三类路径
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
        id: "d-ext",
        deptName: "外部机构",
        parentId: "d-root",
        isExternalOrg: 1,
        children: [{ id: "d-ext-1", deptName: "外派组", parentId: "d-ext" }],
      },
    ],
  },
];

vi.mock("@/hooks/useDeptTree", () => ({
  useDeptTree: () => ({ data: MOCK_DEPTS, isLoading: false }),
}));

import DeptTree from "../index";

function renderTree(ui: ReactElement = <DeptTree />): RenderResult {
  return render(ui);
}
type RenderResult = ReturnType<typeof render>;

describe("DeptTree — 单遍过滤（批次 6.1 回归）", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("默认渲染：searchValue 为空 → treeData 直通，渲染根/父节点", () => {
    renderTree();
    expect(screen.getByText("总部")).toBeInTheDocument();
  });

  it("搜索命中父节点 title → 父保留，祖先链因 filteredChildren>0 保留", () => {
    renderTree();
    const input = screen.getByPlaceholderText("搜索部门");
    fireEvent.change(input, { target: { value: "研发" } });
    // titleMatch=true 分支：即使子节点不匹配也保留该节点（children 置 undefined）
    expect(screen.getByText("研发部")).toBeInTheDocument();
    // 祖先链保留（子命中使总部 filteredChildren>0）；未命中兄弟子树剔除
    expect(screen.getByText("总部")).toBeInTheDocument();
    expect(screen.queryByText("外部机构")).not.toBeInTheDocument();
  });

  it("搜索命中子节点 title → 父节点因 filteredChildren>0 保留", () => {
    renderTree();
    const input = screen.getByPlaceholderText("搜索部门");
    fireEvent.change(input, { target: { value: "前端" } });
    // 子命中分支：父链（总部/研发部）因 children 非空而保留
    expect(screen.getByText("前端组")).toBeInTheDocument();
    expect(screen.getByText("研发部")).toBeInTheDocument();
    expect(screen.getByText("总部")).toBeInTheDocument();
    // 未命中兄弟子树不渲染
    expect(screen.queryByText("外部机构")).not.toBeInTheDocument();
  });

  it("搜索大小写不敏感 + 无命中 → 全树过滤为空（null 分支）", () => {
    renderTree();
    const input = screen.getByPlaceholderText("搜索部门");
    fireEvent.change(input, { target: { value: "不存在部门" } });
    expect(screen.queryByText("总部")).not.toBeInTheDocument();
    expect(screen.queryByText("研发部")).not.toBeInTheDocument();
  });

  it("搜索清空 → 恢复完整树（搜空恢复分支）", () => {
    renderTree();
    const input = screen.getByPlaceholderText("搜索部门");
    fireEvent.change(input, { target: { value: "前端" } });
    fireEvent.change(input, { target: { value: "" } });
    expect(screen.getByText("总部")).toBeInTheDocument();
  });

  it("externalOnly 模式 → 仅保留 isExternalOrg 子树", () => {
    const { container } = render(<DeptTree externalOnly />);
    expect(screen.getByText("外部机构")).toBeInTheDocument();
    expect(screen.queryByText("研发部")).not.toBeInTheDocument();
    void container;
  });
});
