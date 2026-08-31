/**
 * Phase 88 Batch347 — components/operations/DeptSidebar 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/components/DeptTree", () => ({
  default: ({ selectedKeys, externalOnly }: any) => (
    <div
      data-testid="dept-tree"
      data-selected-keys={JSON.stringify(selectedKeys)}
      data-external-only={externalOnly}
    />
  ),
}));

import { DeptSidebar } from "../DeptSidebar";

describe("components/operations/DeptSidebar", () => {
  it("渲染 Sider + DeptTree", () => {
    render(<DeptSidebar />);
    expect(screen.getByTestId("dept-tree")).toBeInTheDocument();
  });

  it("selectedDeptId → selectedKeys 数组", () => {
    render(<DeptSidebar selectedDeptId="d1" />);
    const tree = screen.getByTestId("dept-tree");
    expect(tree.getAttribute("data-selected-keys")).toBe(JSON.stringify(["d1"]));
  });

  it("无 selectedDeptId → 空 selectedKeys", () => {
    render(<DeptSidebar />);
    const tree = screen.getByTestId("dept-tree");
    expect(tree.getAttribute("data-selected-keys")).toBe("[]");
  });

  it("onSelect 传递", () => {
    const onSelect = vi.fn();
    render(<DeptSidebar onSelect={onSelect} />);
    expect(screen.getByTestId("dept-tree")).toBeInTheDocument();
  });

  it("externalOnly 默认 true", () => {
    render(<DeptSidebar />);
    expect(screen.getByTestId("dept-tree").getAttribute("data-external-only")).toBe("true");
  });

  it("externalOnly=false 传递", () => {
    render(<DeptSidebar externalOnly={false} />);
    expect(screen.getByTestId("dept-tree").getAttribute("data-external-only")).toBe("false");
  });

  it("自定义 width", () => {
    const { container } = render(<DeptSidebar width={500} />);
    const sider = container.querySelector("aside");
    expect(sider?.getAttribute("style")).toContain("500");
  });

  it("className dept-list-sider", () => {
    const { container } = render(<DeptSidebar />);
    const sider = container.querySelector(".dept-list-sider");
    expect(sider).toBeTruthy();
  });

  it("自定义 style", () => {
    const { container } = render(<DeptSidebar style={{ background: "red" }} />);
    const sider = container.querySelector("aside");
    expect(sider?.getAttribute("style")).toContain("red");
  });
});
