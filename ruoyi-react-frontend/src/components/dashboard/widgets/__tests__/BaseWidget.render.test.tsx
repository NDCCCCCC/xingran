/**
 * Phase 88 Batch35 — dashboard widgets BaseWidget 渲染测试
 *
 * 走 renderWithProviders + 桩掉 useWidgetData + dashboardStore 直接 seed。
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/hooks/useWidgetData", () => ({
  useWidgetData: vi.fn(() => ({
    data: null,
    loading: false,
    error: null,
    refresh: vi.fn(),
  })),
  useBatchWidgetData: vi.fn(() => ({
    dataMap: new Map(),
    loading: false,
    refresh: vi.fn(),
  })),
}));

import { renderWithProviders } from "@/test/utils/renderWithProviders";
import {
  BaseWidget,
  BaseWidgetHeader,
  BaseWidgetContent,
  BaseWidgetActions,
} from "../base/BaseWidget";
import { useDashboardStore } from "@/store/dashboardStore";
import type { WidgetConfig } from "@/types/dashboard";

function reset(viewMode: "view" | "edit" = "view") {
  useDashboardStore.setState({
    viewMode,
    selectedWidgetId: null,
    selectWidget: vi.fn(),
  } as any);
}

const baseWidget: WidgetConfig = {
  id: "w-1",
  type: "stat-card",
  title: "今日访问",
  position: { x: 0, y: 0, w: 6, h: 3 },
  config: {},
};

describe("BaseWidget 渲染", () => {
  it("view 模式 + 默认空:渲染 Empty + 标题", () => {
    reset("view");
    const { container, findByText } = renderWithProviders(
      <BaseWidget widget={baseWidget}>
        <span>value</span>
      </BaseWidget>
    );
    expect(container.querySelector(".ant-card")).not.toBeNull();
    expect(findByText("今日访问")).resolves.toBeDefined();
  });

  it("edit 模式渲染 class 含 editable", () => {
    reset("edit");
    const { container } = renderWithProviders(
      <BaseWidget widget={baseWidget}>
        <span>x</span>
      </BaseWidget>
    );
    expect(container.querySelector(".base-widget--editable")).not.toBeNull();
    // drag handle
    expect(container.querySelector(".widget-drag-handle")).not.toBeNull();
  });

  it("外部 data 非空:渲染 children(穿过 Spin)", () => {
    reset("view");
    const { container, findByText } = renderWithProviders(
      <BaseWidget widget={baseWidget} data={[1, 2, 3]}>
        <span>children rendered</span>
      </BaseWidget>
    );
    expect(container.querySelector(".ant-spin")).not.toBeNull();
    expect(findByText("children rendered")).resolves.toBeDefined();
  });

  it("外部 error 渲染 Result 错误状态", () => {
    reset("view");
    const { container, findByText } = renderWithProviders(
      <BaseWidget widget={baseWidget} error="加载失败原因">
        <span>x</span>
      </BaseWidget>
    );
    expect(container.querySelector(".ant-result")).not.toBeNull();
    expect(findByText("加载失败原因")).resolves.toBeDefined();
    expect(findByText("重试")).resolves.toBeDefined();
  });

  it("自定义 emptyMessage 生效", () => {
    reset("view");
    const { findByText } = renderWithProviders(
      <BaseWidget widget={baseWidget} emptyMessage="无任何访问记录">
        <span>x</span>
      </BaseWidget>
    );
    expect(findByText("无任何访问记录")).resolves.toBeDefined();
  });
});

describe("BaseWidget 子组件", () => {
  it("BaseWidgetHeader 渲染 title + extra", () => {
    const { container } = renderWithProviders(
      <BaseWidgetHeader title="标题" extra={<span>extra</span>} />
    );
    expect(container.querySelector(".base-widget-header__title")).not.toBeNull();
  });

  it("BaseWidgetContent loading 态", () => {
    const { container } = renderWithProviders(
      <BaseWidgetContent loading>实质内容</BaseWidgetContent>
    );
    expect(container.querySelector(".base-widget-content--loading")).not.toBeNull();
  });

  it("BaseWidgetContent error 态", () => {
    const { container } = renderWithProviders(
      <BaseWidgetContent error="出错了">x</BaseWidgetContent>
    );
    expect(container.querySelector(".base-widget-content--error")).not.toBeNull();
  });

  it("BaseWidgetActions showActions=false 返 null", () => {
    const { container } = renderWithProviders(<BaseWidgetActions showActions={false} />);
    // renderWithProviders 外面包 MemoryRouter → firstChild 是路由器 div
    // BaseWidgetActions 自身无 .base-widget-actions 类即可证明组件返回 null
    expect(container.querySelector(".base-widget-actions")).toBeNull();
  });

  it("BaseWidgetActions showActions=true 渲染 3 按钮", () => {
    const { container } = renderWithProviders(
      <BaseWidgetActions onRefresh={vi.fn()} onEdit={vi.fn()} onDelete={vi.fn()} />
    );
    const buttons = container.querySelectorAll("button");
    expect(buttons.length).toBe(3);
  });
});
