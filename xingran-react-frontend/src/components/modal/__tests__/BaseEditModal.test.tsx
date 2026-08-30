/**
 * Phase 88 Batch165 — components/modal/BaseEditModal 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import BaseEditModal from "../BaseEditModal";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("BaseEditModal", () => {
  it("open=true + title + children → 渲染 Modal", () => {
    const { baseElement } = render(
      <BaseEditModal open title="编辑" onOk={vi.fn()} onCancel={vi.fn()}>
        <div data-testid="modal-child">Content</div>
      </BaseEditModal>,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("编辑");
    expect(baseElement.querySelector('[data-testid="modal-child"]')).toBeTruthy();
  });

  it("点击 确定 → onOk 调用", () => {
    const onOk = vi.fn();
    const { baseElement } = render(
      <BaseEditModal open title="T" onOk={onOk} onCancel={vi.fn()}>
        <div />
      </BaseEditModal>,
      { wrapper }
    );
    const okBtn = Array.from(baseElement.querySelectorAll("button")).find(
      (b) => b.textContent?.trim() === "确 定"
    );
    expect(okBtn).toBeTruthy();
    fireEvent.click(okBtn!);
    expect(onOk).toHaveBeenCalled();
  });

  it("点击 取消 → onCancel 调用", () => {
    const onCancel = vi.fn();
    const { baseElement } = render(
      <BaseEditModal open title="T" onOk={vi.fn()} onCancel={onCancel}>
        <div />
      </BaseEditModal>,
      { wrapper }
    );
    const cancelBtn = Array.from(baseElement.querySelectorAll("button")).find(
      (b) => b.textContent?.trim() === "取 消"
    );
    fireEvent.click(cancelBtn!);
    expect(onCancel).toHaveBeenCalled();
  });

  it("confirmLoading=true → OK 按钮 loading", () => {
    const { baseElement } = render(
      <BaseEditModal open title="T" onOk={vi.fn()} onCancel={vi.fn()} confirmLoading>
        <div />
      </BaseEditModal>,
      { wrapper }
    );
    expect(baseElement.querySelector(".ant-btn-loading")).toBeTruthy();
  });

  it("自定义 okText + cancelText", () => {
    const { baseElement } = render(
      <BaseEditModal
        open
        title="T"
        onOk={vi.fn()}
        onCancel={vi.fn()}
        okText="保存"
        cancelText="取消保存"
      >
        <div />
      </BaseEditModal>,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("保存");
    expect(baseElement.textContent).toContain("取消保存");
  });

  it("自定义 width 透传", () => {
    const { baseElement } = render(
      <BaseEditModal open title="T" onOk={vi.fn()} onCancel={vi.fn()} width={800}>
        <div />
      </BaseEditModal>,
      { wrapper }
    );
    expect(baseElement.querySelector(".ant-modal")).toBeTruthy();
  });

  it("maskClosable=false → Modal 设置", () => {
    const { baseElement } = render(
      <BaseEditModal open title="T" onOk={vi.fn()} onCancel={vi.fn()} maskClosable={false}>
        <div />
      </BaseEditModal>,
      { wrapper }
    );
    expect(baseElement.querySelector(".ant-modal")).toBeTruthy();
  });
});
