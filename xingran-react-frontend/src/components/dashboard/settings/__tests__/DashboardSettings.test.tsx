/**
 * Phase 88 Batch121 — components/dashboard/settings/DashboardSettings 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent, act } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

let storeState: Record<string, any> = {};
vi.mock("@/store/dashboardStore", () => ({
  useDashboardStore: () => storeState,
}));

vi.mock("@/components/dashboard/settings/DashboardScopeSelector", () => ({
  default: ({ value, onChange }: any) => (
    <div data-testid="scope-selector">
      <button onClick={() => onChange?.({ scope: "public" })}>change-scope</button>
      <span>{value?.scope}</span>
    </div>
  ),
}));

vi.mock("@/components/dashboard/settings/RefreshIntervalSelector", () => ({
  default: () => <div data-testid="refresh-selector" />,
}));

import DashboardSettings from "../DashboardSettings";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("DashboardSettings", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    storeState = {
      currentDashboard: {
        id: "d1",
        name: "主仪表盘",
        description: "测试",
        refreshInterval: 300,
        scope: "private",
        createdBy: "admin",
        createdAt: "2026-01-01",
        updatedAt: "2026-08-01",
        isDefault: false,
        isTemplate: false,
      },
      updateDashboard: vi.fn(() => Promise.resolve()),
    };
  });

  it("currentDashboard=null → 不渲染仪表盘信息", () => {
    storeState.currentDashboard = null;
    const { baseElement } = render(<DashboardSettings visible onClose={vi.fn()} />, { wrapper });
    expect(baseElement.textContent).not.toContain("基本信息");
  });

  it("visible=true + currentDashboard → 渲染抽屉 + 表单字段", async () => {
    const { baseElement } = render(<DashboardSettings visible onClose={vi.fn()} />, { wrapper });
    await act(async () => {});
    expect(baseElement.textContent).toContain("仪表盘设置");
    expect(baseElement.textContent).toContain("基本信息");
    expect(baseElement.textContent).toContain("权限设置");
    expect(baseElement.textContent).toContain("刷新设置");
    expect(baseElement.textContent).toContain("仪表盘信息");
    expect(baseElement.textContent).toContain("d1"); // id
  });

  it("isDefault=true → 显示默认仪表盘标记", async () => {
    storeState.currentDashboard = { ...storeState.currentDashboard, isDefault: true };
    const { baseElement } = render(<DashboardSettings visible onClose={vi.fn()} />, { wrapper });
    await act(async () => {});
    expect(baseElement.textContent).toContain("默认仪表盘");
  });

  it("isTemplate=true → 显示模板标记", async () => {
    storeState.currentDashboard = { ...storeState.currentDashboard, isTemplate: true };
    const { baseElement } = render(<DashboardSettings visible onClose={vi.fn()} />, { wrapper });
    await act(async () => {});
    expect(baseElement.textContent).toContain("模板");
  });

  it("点击 保存 + 校验通过 → 调用 updateDashboard + onClose", async () => {
    const onClose = vi.fn();
    const { baseElement, getByText } = render(<DashboardSettings visible onClose={onClose} />, {
      wrapper,
    });
    await act(async () => {});
    await act(async () => {
      fireEvent.click(getByText("保存"));
    });
    expect(storeState.updateDashboard).toHaveBeenCalledWith("d1", expect.any(Object));
    expect(onClose).toHaveBeenCalled();
  });

  it("点击 取消 → onClose + resetFields", async () => {
    const onClose = vi.fn();
    const { baseElement, getByText } = render(<DashboardSettings visible onClose={onClose} />, {
      wrapper,
    });
    await act(async () => {});
    fireEvent.click(getByText("取消"));
    expect(onClose).toHaveBeenCalled();
  });

  it("scope 选择变化 → setScopeConfig", async () => {
    const { baseElement, getByText } = render(<DashboardSettings visible onClose={vi.fn()} />, {
      wrapper,
    });
    await act(async () => {});
    fireEvent.click(getByText("change-scope"));
    expect(baseElement.querySelector('[data-testid="scope-selector"]')?.textContent).toContain(
      "public"
    );
  });

  it("保存失败 → 错误提示 + 不调 onClose", async () => {
    storeState.updateDashboard = vi.fn(() => Promise.reject(new Error("保存失败")));
    const errSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const onClose = vi.fn();
    const { getByText } = render(<DashboardSettings visible onClose={onClose} />, { wrapper });
    await act(async () => {});
    await act(async () => {
      fireEvent.click(getByText("保存"));
    });
    expect(errSpy).toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
    errSpy.mockRestore();
  });
});
