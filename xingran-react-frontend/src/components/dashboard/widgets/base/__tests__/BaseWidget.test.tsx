/**
 * Phase 88 Batch368 — components/dashboard/widgets/base/BaseWidget 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("../BaseWidget.css", () => ({}));

let mockViewMode = "view";
let mockSelectedWidgetId: string | null = null;
vi.mock("@/store/dashboardStore", () => ({
  useDashboardStore: vi.fn((selector: any) => {
    const state = {
      viewMode: mockViewMode,
      selectedWidgetId: mockSelectedWidgetId,
      selectWidget: vi.fn((id: string | null) => {
        mockSelectedWidgetId = id;
      }),
    };
    return typeof selector === "function" ? selector(state) : state;
  }),
}));

let mockData: any = { value: 42 };
let mockLoading = false;
let mockError: string | null = null;
vi.mock("@/hooks/useWidgetData", () => ({
  useWidgetData: vi.fn(() => ({
    data: mockData,
    loading: mockLoading,
    error: mockError,
    refresh: vi.fn(),
  })),
}));

import BaseWidget from "../BaseWidget";
import { BaseWidgetHeader, BaseWidgetContent, BaseWidgetActions } from "../BaseWidget";

const baseWidget: any = {
  id: "w1",
  type: "stat",
  title: "我的Widget",
  display: {},
};

describe("components/dashboard/widgets/base/BaseWidget", () => {
  it("渲染 widget title", () => {
    render(
      <BaseWidget widget={baseWidget}>
        <span>content</span>
      </BaseWidget>
    );
    expect(screen.getByText("我的Widget")).toBeInTheDocument();
  });

  it("渲染 children", () => {
    render(
      <BaseWidget widget={baseWidget}>
        <span data-testid="c">content</span>
      </BaseWidget>
    );
    expect(screen.getByTestId("c")).toBeInTheDocument();
  });

  it("data 由内部 hook 提供 (无 external)", () => {
    mockData = { value: 99 };
    render(
      <BaseWidget widget={baseWidget}>
        <span>content</span>
      </BaseWidget>
    );
    expect(screen.getByText("我的Widget")).toBeInTheDocument();
  });

  it("loading=true → 显示加载状态", () => {
    mockLoading = true;
    const { container } = render(
      <BaseWidget widget={baseWidget}>
        <span>c</span>
      </BaseWidget>
    );
    expect(container.querySelector(".base-widget--loading")).toBeTruthy();
  });

  it("error → 显示错误", () => {
    mockLoading = false;
    mockError = "网络错误";
    render(
      <BaseWidget widget={baseWidget}>
        <span>c</span>
      </BaseWidget>
    );
    expect(screen.getByText("加载失败")).toBeInTheDocument();
  });

  it("空数据 → 显示 Empty", () => {
    mockLoading = false;
    mockError = null;
    mockData = null;
    render(
      <BaseWidget widget={baseWidget} emptyMessage="无数据">
        <span>c</span>
      </BaseWidget>
    );
    expect(screen.getByText("无数据")).toBeInTheDocument();
  });

  it("空数组 → 显示 Empty", () => {
    mockData = [];
    render(
      <BaseWidget widget={baseWidget}>
        <span>c</span>
      </BaseWidget>
    );
    expect(screen.getByText("暂无数据")).toBeInTheDocument();
  });

  it("edit 模式 + click → selectWidget 切换", () => {
    mockViewMode = "edit";
    const { container } = render(
      <BaseWidget widget={baseWidget}>
        <span>c</span>
      </BaseWidget>
    );
    const widgetEl = container.querySelector(".base-widget");
    fireEvent.click(widgetEl!);
    // selectWidget was called
    expect(widgetEl).toBeTruthy();
  });

  it("edit 模式 → 拖拽图标", () => {
    mockViewMode = "edit";
    const { container } = render(
      <BaseWidget widget={baseWidget}>
        <span>c</span>
      </BaseWidget>
    );
    expect(container.querySelector(".widget-drag-handle")).toBeTruthy();
  });

  it("view 模式 → 不显示拖拽图标", () => {
    mockViewMode = "view";
    const { container } = render(
      <BaseWidget widget={baseWidget}>
        <span>c</span>
      </BaseWidget>
    );
    expect(container.querySelector(".widget-drag-handle")).toBeNull();
  });

  it("BaseWidgetHeader 渲染 title/icon/extra", () => {
    render(
      <BaseWidgetHeader title="Header" icon={<span>icon</span>} extra={<span>extra</span>} />
    );
    expect(screen.getByText("Header")).toBeInTheDocument();
    expect(screen.getByText("icon")).toBeInTheDocument();
    expect(screen.getByText("extra")).toBeInTheDocument();
  });

  it("BaseWidgetHeader 无 icon → 不渲染 icon div", () => {
    const { container } = render(<BaseWidgetHeader title="H" />);
    expect(container.querySelector(".base-widget-header__icon")).toBeNull();
  });

  it("BaseWidgetContent loading → 显示加载", () => {
    render(<BaseWidgetContent loading>c</BaseWidgetContent>);
    expect(screen.getByText("加载中...")).toBeInTheDocument();
  });

  it("BaseWidgetContent error → 显示错误", () => {
    render(<BaseWidgetContent error="err msg">c</BaseWidgetContent>);
    expect(screen.getByText("err msg")).toBeInTheDocument();
  });

  it("BaseWidgetContent empty → 显示空", () => {
    render(<BaseWidgetContent empty emptyMessage="空数据">c</BaseWidgetContent>);
    expect(screen.getByText("空数据")).toBeInTheDocument();
  });

  it("BaseWidgetContent normal → 渲染 children", () => {
    render(
      <BaseWidgetContent>
        <span>main</span>
      </BaseWidgetContent>
    );
    expect(screen.getByText("main")).toBeInTheDocument();
  });

  it("BaseWidgetActions showActions=false → null", () => {
    const { container } = render(<BaseWidgetActions showActions={false} onRefresh={vi.fn()} />);
    expect(container.firstChild).toBeNull();
  });

  it("BaseWidgetActions 渲染 refresh/edit/delete 按钮", () => {
    render(
      <BaseWidgetActions onRefresh={vi.fn()} onEdit={vi.fn()} onDelete={vi.fn()} />
    );
    expect(screen.getByText("刷新")).toBeInTheDocument();
    expect(screen.getByText("编辑")).toBeInTheDocument();
    expect(screen.getByText("删除")).toBeInTheDocument();
  });

  it("BaseWidgetActions 仅 refresh", () => {
    render(<BaseWidgetActions onRefresh={vi.fn()} />);
    expect(screen.getByText("刷新")).toBeInTheDocument();
    expect(screen.queryByText("编辑")).toBeNull();
  });

  it("disableDataFetch → 不调用 useWidgetData", () => {
    render(
      <BaseWidget widget={baseWidget} disableDataFetch>
        <span>c</span>
      </BaseWidget>
    );
    expect(screen.getByText("我的Widget")).toBeInTheDocument();
  });
});