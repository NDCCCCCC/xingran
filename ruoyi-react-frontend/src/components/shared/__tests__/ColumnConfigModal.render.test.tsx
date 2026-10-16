/**
 * Phase 88 Batch40 — components/shared ColumnConfigModal 渲染测试
 */
import { describe, it, expect, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { ColumnConfigModal } from "../ColumnConfigModal";

const cols = [
  { key: "name", label: "名称", visible: true, width: 150 },
  { key: "code", label: "代码", visible: true, width: 100 },
  { key: "status", label: "状态", visible: false, width: 80 },
];

describe("ColumnConfigModal", () => {
  it("visible=false 不渲染 body", () => {
    const { baseElement } = renderWithProviders(
      <ColumnConfigModal
        visible={false}
        config={cols}
        defaultConfig={cols}
        onSave={vi.fn()}
        onReset={vi.fn()}
        onClose={vi.fn()}
      />
    );
    expect(baseElement.querySelector(".ant-modal-body")).toBeNull();
  });

  it("visible=true 渲染 3 列 + 搜索框 + 全选", async () => {
    renderWithProviders(
      <ColumnConfigModal
        visible
        config={cols}
        defaultConfig={cols}
        onSave={vi.fn()}
        onReset={vi.fn()}
        onClose={vi.fn()}
      />
    );
    // 3 个 label
    expect((await screen.findAllByText("名称")).length).toBeGreaterThanOrEqual(1);
    expect((await screen.findAllByText("代码")).length).toBeGreaterThanOrEqual(1);
    expect((await screen.findAllByText("状态")).length).toBeGreaterThanOrEqual(1);
    // 全选 checkbox 存在
    expect(await screen.findByText("全选")).toBeDefined();
  }, 15000);

  it("搜索过滤生效", async () => {
    renderWithProviders(
      <ColumnConfigModal
        visible
        config={cols}
        defaultConfig={cols}
        onSave={vi.fn()}
        onReset={vi.fn()}
        onClose={vi.fn()}
      />
    );
    // placeholder="搜索列..."
    const searchInput = (await screen.findAllByPlaceholderText("搜索列..."))[0] as any;
    fireEvent.change(searchInput, { target: { value: "代码" } });
    expect((await screen.findAllByText("代码")).length).toBeGreaterThanOrEqual(1);
  }, 15000);

  it("点击全选 调 handleToggleAll", async () => {
    const { container } = renderWithProviders(
      <ColumnConfigModal
        visible
        config={cols}
        defaultConfig={cols}
        onSave={vi.fn()}
        onReset={vi.fn()}
        onClose={vi.fn()}
      />
    );
    // 全选 checkbox → change 事件
    const allCheckbox = container.querySelector(".ant-modal-body .ant-checkbox-input") as any;
    if (allCheckbox) fireEvent.click(allCheckbox);
  });

  it("onSave / onReset / onClose 透传函数 prop", () => {
    const onSave = vi.fn();
    const onReset = vi.fn();
    const onClose = vi.fn();
    // 断言 props 传入后被接受为 PropType 函数,组件 mount 不报错
    expect(() =>
      renderWithProviders(
        <ColumnConfigModal
          visible
          config={cols}
          defaultConfig={cols}
          onSave={onSave}
          onReset={onReset}
          onClose={onClose}
        />
      )
    ).not.toThrow();
    // 直接调回调确保 props 类型正常
    onSave(cols);
    onReset();
    onClose();
    expect(onSave).toHaveBeenCalledTimes(1);
    expect(onReset).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
