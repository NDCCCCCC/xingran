/**
 * Phase 88 Batch350 — components/table/AssetRow 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import AssetRow from "../AssetRow";

describe("components/table/AssetRow", () => {
  const sampleAsset: unknown = { id: "a1", name: "Asset 1" };

  it("有 onEdit 时渲染编辑/删除按钮", () => {
    render(<AssetRow record={sampleAsset} onEdit={vi.fn()} />);
    expect(screen.getByText("编辑")).toBeInTheDocument();
    expect(screen.getByText("删除")).toBeInTheDocument();
  });

  it("点击编辑 → onEdit(record)", () => {
    const onEdit = vi.fn();
    render(<AssetRow record={sampleAsset} onEdit={onEdit} />);
    fireEvent.click(screen.getByText("编辑"));
    expect(onEdit).toHaveBeenCalledWith(sampleAsset);
  });

  it("点击删除 → onDelete(id)", () => {
    const onDelete = vi.fn();
    render(<AssetRow record={sampleAsset} onDelete={onDelete} />);
    fireEvent.click(screen.getByText("删除"));
    expect(onDelete).toHaveBeenCalledWith("a1");
  });

  it("无 onEdit 时隐藏编辑入口（STOP-06）且不抛错", () => {
    const { container } = render(<AssetRow record={sampleAsset} />);
    expect(screen.queryByText("编辑")).not.toBeInTheDocument();
    expect(container.querySelector(".ant-btn-dangerous")).toBeInTheDocument();
  });

  it("无 onDelete 不抛错", () => {
    expect(() => render(<AssetRow record={sampleAsset} />)).not.toThrow();
    fireEvent.click(screen.getByText("删除"));
  });

  it("displayName 正确", () => {
    expect(AssetRow.displayName).toBe("AssetRow");
  });

  it("memo 包裹", () => {
    expect((AssetRow as unknown).$$typeof).toBeDefined();
  });

  it("删除按钮有 danger 属性", () => {
    const { container } = render(<AssetRow record={sampleAsset} />);
    const deleteBtn = container.querySelector(".ant-btn-dangerous");
    expect(deleteBtn).toBeTruthy();
  });
});
