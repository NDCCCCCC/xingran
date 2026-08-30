/**
 * Phase 88 Batch118 — components/dashboard/layout/LayoutToolbar 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent, act } from "@testing-library/react";
import { App as AntdApp, Modal } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const storeState: Record<string, any> = {
  viewMode: "view",
  hasUnsavedChanges: false,
  saveCurrentDashboard: vi.fn(() => Promise.resolve()),
  resetCurrentDashboard: vi.fn(() => Promise.resolve()),
  currentDashboard: null,
  setViewMode: vi.fn(),
};

vi.mock("@/store/dashboardStore", () => ({
  useDashboardStore: () => storeState,
}));

const navigateMock = vi.fn();
vi.mock("react-router-dom", () => ({
  useNavigate: () => navigateMock,
}));

import { LayoutToolbar } from "../LayoutToolbar";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("LayoutToolbar", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    storeState.viewMode = "view";
    storeState.hasUnsavedChanges = false;
    storeState.currentDashboard = null;
    storeState.saveCurrentDashboard = vi.fn(() => Promise.resolve());
    storeState.resetCurrentDashboard = vi.fn(() => Promise.resolve());
    storeState.setViewMode = vi.fn();
  });

  it("view 模式 → 显示 编辑/设置 按钮", () => {
    storeState.viewMode = "view";
    const { container } = render(<LayoutToolbar />, { wrapper });
    expect(container.textContent).toContain("编辑");
    expect(container.textContent).toContain("设置");
  });

  it("edit 模式 + hasUnsavedChanges=false → 显示 保存/预览 + 保存按钮 disabled", () => {
    storeState.viewMode = "edit";
    storeState.hasUnsavedChanges = false;
    const { container } = render(<LayoutToolbar />, { wrapper });
    expect(container.textContent).toContain("保存");
    expect(container.textContent).toContain("预览");
  });

  it("edit 模式 + hasUnsavedChanges=true → 显示 重置按钮 + 保存 enabled", () => {
    storeState.viewMode = "edit";
    storeState.hasUnsavedChanges = true;
    const { container, getByText } = render(<LayoutToolbar />, { wrapper });
    expect(container.textContent).toContain("重置");
    const saveBtn = getByText("保存").closest("button");
    expect(saveBtn?.disabled).toBe(false);
  });

  it("点击 编辑按钮(view→edit) → 调用 setViewMode('edit')", () => {
    storeState.viewMode = "view";
    const { getByText } = render(<LayoutToolbar />, { wrapper });
    fireEvent.click(getByText("编辑"));
    expect(storeState.setViewMode).toHaveBeenCalledWith("edit");
  });

  it("点击 编辑按钮 + dashboardId → 导航到 ?mode=edit", () => {
    storeState.viewMode = "view";
    const { getByText } = render(<LayoutToolbar dashboardId="d1" />, { wrapper });
    fireEvent.click(getByText("编辑"));
    expect(navigateMock).toHaveBeenCalledWith(expect.stringContaining("?mode=edit"));
  });

  it("点击 保存 + 无更改 → message.info + 不调 save", () => {
    storeState.viewMode = "edit";
    storeState.hasUnsavedChanges = false;
    const { getByText } = render(<LayoutToolbar />, { wrapper });
    fireEvent.click(getByText("保存"));
    expect(storeState.saveCurrentDashboard).not.toHaveBeenCalled();
  });

  it("点击 保存 + 有更改 → 调用 saveCurrentDashboard", async () => {
    storeState.viewMode = "edit";
    storeState.hasUnsavedChanges = true;
    const { getByText } = render(<LayoutToolbar />, { wrapper });
    await act(async () => {
      fireEvent.click(getByText("保存"));
    });
    expect(storeState.saveCurrentDashboard).toHaveBeenCalled();
  });

  it("点击 重置 + Modal.confirm 取消 → 不调 reset", async () => {
    storeState.viewMode = "edit";
    storeState.hasUnsavedChanges = true;
    const confirmSpy = vi.spyOn(Modal, "confirm");
    const origConfirm = Modal.confirm;
    Modal.confirm = vi.fn(({ onOk }) => {
      // simulate cancel - don't call onOk
      return { destroy: vi.fn() } as any;
    }) as any;
    const { getByText } = render(<LayoutToolbar />, { wrapper });
    fireEvent.click(getByText("重置"));
    expect(storeState.resetCurrentDashboard).not.toHaveBeenCalled();
    Modal.confirm = origConfirm;
    confirmSpy.mockRestore();
  });

  it("点击 添加Widget + onAddWidget → 调用回调", () => {
    storeState.viewMode = "edit";
    const onAdd = vi.fn();
    const { getByText } = render(<LayoutToolbar onAddWidget={onAdd} />, { wrapper });
    fireEvent.click(getByText("添加Widget"));
    expect(onAdd).toHaveBeenCalled();
  });

  it("currentDashboard.name → 显示标题", () => {
    storeState.currentDashboard = { name: "主仪表盘" };
    const { container } = render(<LayoutToolbar />, { wrapper });
    expect(container.textContent).toContain("主仪表盘");
  });

  it("showBackButton=true + hasUnsavedChanges=false → 直接 navigate", () => {
    storeState.hasUnsavedChanges = false;
    const { container } = render(<LayoutToolbar showBackButton />, { wrapper });
    fireEvent.click(container.querySelector("button")!);
    expect(navigateMock).toHaveBeenCalled();
  });

  it("showBackButton=true + hasUnsavedChanges=true → Modal.confirm", () => {
    storeState.hasUnsavedChanges = true;
    const origConfirm = Modal.confirm;
    Modal.confirm = vi.fn(({ onOk }) => {
      onOk();
      return { destroy: vi.fn() } as any;
    }) as any;
    const { container } = render(<LayoutToolbar showBackButton />, { wrapper });
    fireEvent.click(container.querySelector("button")!);
    expect(navigateMock).toHaveBeenCalled();
    Modal.confirm = origConfirm;
  });
});
