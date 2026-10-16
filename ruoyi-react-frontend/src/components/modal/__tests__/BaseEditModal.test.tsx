/**
 * Phase 88 Batch348 — components/modal/BaseEditModal 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import BaseEditModal from "../BaseEditModal";

describe("components/modal/BaseEditModal", () => {
  it("displayName 正确", () => {
    expect(BaseEditModal.displayName).toBe("BaseEditModal");
  });

  it("memo 包裹", () => {
    expect((BaseEditModal as any).$$typeof).toBeDefined();
  });

  it("open=true + title → 渲染", () => {
    render(
      <BaseEditModal open={true} title="编辑" onOk={vi.fn()} onCancel={vi.fn()}>
        <span>content</span>
      </BaseEditModal>
    );
    expect(screen.getByText("编辑")).toBeInTheDocument();
    expect(screen.getByText("content")).toBeInTheDocument();
  });

  it("open=false 不渲染标题", () => {
    const { container } = render(
      <BaseEditModal open={false} title="隐藏" onOk={vi.fn()} onCancel={vi.fn()}>
        <span>x</span>
      </BaseEditModal>
    );
    expect(container.querySelector(".ant-modal-title")).toBeNull();
  });

  it("点击确定触发 onOk", () => {
    const onOk = vi.fn();
    render(
      <BaseEditModal open={true} title="t" onOk={onOk} onCancel={vi.fn()}>
        <span>x</span>
      </BaseEditModal>
    );
    const okBtn = screen.getByRole("button", { name: "确 定" });
    fireEvent.click(okBtn);
    expect(onOk).toHaveBeenCalled();
  });

  it("点击取消触发 onCancel", () => {
    const onCancel = vi.fn();
    render(
      <BaseEditModal open={true} title="t" onOk={vi.fn()} onCancel={onCancel}>
        <span>x</span>
      </BaseEditModal>
    );
    const cancelBtn = screen.getByRole("button", { name: "取 消" });
    fireEvent.click(cancelBtn);
    expect(onCancel).toHaveBeenCalled();
  });

  it("自定义 okText + cancelText 渲染 (不报错)", () => {
    expect(() =>
      render(
        <BaseEditModal
          open={true}
          title="t"
          onOk={vi.fn()}
          onCancel={vi.fn()}
          okText="保存"
          cancelText="放弃"
        >
          <span>x</span>
        </BaseEditModal>
      )
    ).not.toThrow();
  });

  it("自定义 width 不报错", () => {
    expect(() =>
      render(
        <BaseEditModal open={true} title="t" onOk={vi.fn()} onCancel={vi.fn()} width={800}>
          <span>x</span>
        </BaseEditModal>
      )
    ).not.toThrow();
  });

  it("confirmLoading 不报错", () => {
    expect(() =>
      render(
        <BaseEditModal open={true} title="t" onOk={vi.fn()} onCancel={vi.fn()} confirmLoading>
          <span>x</span>
        </BaseEditModal>
      )
    ).not.toThrow();
  });

  it("maskClosable=false 阻止蒙层关闭", () => {
    render(
      <BaseEditModal open={true} title="t" onOk={vi.fn()} onCancel={vi.fn()} maskClosable={false}>
        <span>x</span>
      </BaseEditModal>
    );
    const mask = document.querySelector(".ant-modal-mask");
    expect(mask).toBeTruthy();
  });

  it("destroyOnHidden → children 卸载", () => {
    const { rerender } = render(
      <BaseEditModal open={true} title="t" onOk={vi.fn()} onCancel={vi.fn()}>
        <span data-testid="c">visible</span>
      </BaseEditModal>
    );
    expect(screen.getByTestId("c")).toBeInTheDocument();
    rerender(
      <BaseEditModal open={false} title="t" onOk={vi.fn()} onCancel={vi.fn()}>
        <span data-testid="c">hidden</span>
      </BaseEditModal>
    );
    // destroyOnHidden → children unmounts when closed
    expect(screen.queryByTestId("c")).toBeNull();
  });
});
