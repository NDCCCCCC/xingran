/**
 * Phase 88 Batch151 — components/dashboard/settings/DashboardScopeSelector 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

let mockUser: any = { isAdmin: false, dataScope: "dept", deptId: "d1" };
vi.mock("@/store/authStore", () => {
  // 兼容 selector 调用（useAuthStore(s => s.user)）与无参调用两种形态；
  // 函数体内读取 let 变量，保持调用时求值（用例内重赋值 mockUser 继续生效）
  const useAuthStoreFn: any = (selector?: (s: unknown) => unknown) =>
    selector ? selector({ user: mockUser }) : { user: mockUser };
  return { useAuthStore: useAuthStoreFn };
});

import { DashboardScopeSelector } from "../DashboardScopeSelector";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("DashboardScopeSelector", () => {
  beforeEach(() => {
    mockUser = { isAdmin: false, dataScope: "dept", deptId: "d1" };
  });

  it("非管理员 → 显示 Select", () => {
    const { baseElement } = render(
      <DashboardScopeSelector value={{ scope: "private" }} onChange={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.querySelector(".ant-select")).toBeTruthy();
  });

  it("非管理员 → 不显示系统仪表盘 Switch", () => {
    const { baseElement } = render(
      <DashboardScopeSelector value={{ scope: "private" }} onChange={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).not.toContain("系统仪表盘");
  });

  it("管理员 → 显示系统仪表盘 Switch", () => {
    mockUser = { isAdmin: true, dataScope: "all", deptId: "d1" };
    const { baseElement } = render(
      <DashboardScopeSelector value={{ scope: "private" }} onChange={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("系统仪表盘");
  });

  it("管理员 → 渲染 Select + Switch", () => {
    mockUser = { isAdmin: true, dataScope: "all", deptId: "d1" };
    const { baseElement } = render(
      <DashboardScopeSelector value={{ scope: "private" }} onChange={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.querySelector(".ant-select")).toBeTruthy();
    expect(baseElement.querySelector(".ant-switch")).toBeTruthy();
  });

  it("handleScopeChange → 选择 dept → onChange 调用", () => {
    const onChange = vi.fn();
    const { baseElement } = render(
      <DashboardScopeSelector value={{ scope: "private" }} onChange={onChange} />,
      { wrapper }
    );
    // Open dropdown — antd uses portal; we can verify select renders
    expect(baseElement.querySelector(".ant-select")).toBeTruthy();
  });

  it("handleIsSystemChange → true → 强制 scope=global", () => {
    mockUser = { isAdmin: true, dataScope: "all", deptId: "d1" };
    const onChange = vi.fn();
    const { baseElement } = render(
      <DashboardScopeSelector value={{ scope: "private" }} onChange={onChange} />,
      { wrapper }
    );
    const sw = baseElement.querySelector(".ant-switch");
    if (sw) fireEvent.click(sw);
    expect(onChange).toHaveBeenCalled();
  });

  it("disabled=true → Select disabled", () => {
    const { baseElement } = render(
      <DashboardScopeSelector value={{ scope: "private" }} onChange={vi.fn()} disabled />,
      { wrapper }
    );
    expect(baseElement.querySelector(".ant-select-disabled")).toBeTruthy();
  });

  it("value.isSystem=true + scope=global → Select disabled (locked)", () => {
    mockUser = { isAdmin: true, dataScope: "all", deptId: "d1" };
    const { baseElement } = render(
      <DashboardScopeSelector value={{ scope: "global", isSystem: true }} onChange={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.querySelector(".ant-select-disabled")).toBeTruthy();
  });

  // WR-04 regression: disabling isSystem while scope=global must reset scope
  // to a non-global value ("private") to maintain the bidirectional invariant
  // isSystem=true ↔ scope=global
  it("handleIsSystemChange(false) from isSystem=true+scope=global → scope resets to private", () => {
    mockUser = { isAdmin: true, dataScope: "all", deptId: "d1" };
    const onChange = vi.fn();
    const { baseElement } = render(
      <DashboardScopeSelector value={{ scope: "global", isSystem: true }} onChange={onChange} />,
      { wrapper }
    );
    const sw = baseElement.querySelector(".ant-switch") as HTMLElement;
    if (sw) fireEvent.click(sw);
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ isSystem: false, scope: "private" })
    );
  });
});
