/**
 * Phase 88 Batch119 — components/dashboard/settings/WidgetSelector 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent, act } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/components/dashboard/widgets/configs/widgetRegistry", () => ({
  widgetRegistry: {
    "stat-card": { displayName: "统计卡片", description: "数字", icon: "📊" },
    chart: { displayName: "图表", description: "图", icon: "📈" },
    table: { displayName: "表格", description: "表", icon: "📋" },
    list: { displayName: "列表", description: "列", icon: "📃" },
    progress: { displayName: "进度", description: "进", icon: "⏳" },
  },
  getWidgetTypes: () => ["stat-card", "chart", "table", "list", "progress"],
}));

vi.mock("@/components/dashboard/settings/DataSourceForm", () => ({
  default: () => <div data-testid="data-source-form" />,
}));

vi.mock("@/components/dashboard/settings/DisplayConfigForm", () => ({
  default: () => <div data-testid="display-config-form" />,
}));

import WidgetSelector from "../WidgetSelector";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("WidgetSelector", () => {
  it("visible=true + 未选类型 → 渲染类型卡片列表", () => {
    const { baseElement } = render(
      <WidgetSelector visible onClose={vi.fn()} onSelect={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("添加 Widget");
    expect(baseElement.textContent).toContain("统计卡片");
    expect(baseElement.textContent).toContain("图表");
  });

  it("点击类型卡片 → 切到 config 视图", () => {
    const { baseElement, getByText } = render(
      <WidgetSelector visible onClose={vi.fn()} onSelect={vi.fn()} />,
      { wrapper }
    );
    fireEvent.click(getByText("统计卡片"));
    expect(baseElement.textContent).toContain("Widget类型");
    expect(baseElement.textContent).toContain("标题");
    expect(baseElement.textContent).toContain("数据源配置");
  });

  it("点击 '← 返回选择Widget类型' → 回到类型列表", () => {
    const { baseElement, getByText } = render(
      <WidgetSelector visible onClose={vi.fn()} onSelect={vi.fn()} />,
      { wrapper }
    );
    fireEvent.click(getByText("统计卡片"));
    expect(baseElement.textContent).toContain("Widget类型");
    fireEvent.click(getByText("← 返回选择Widget类型"));
    expect(baseElement.textContent).toContain("统计卡片");
  });

  it("visible=true + editingWidgetId → 标题显示 '编辑 Widget'", () => {
    const { baseElement } = render(
      <WidgetSelector
        visible
        editingWidgetId="w1"
        editingWidget={{
          id: "w1",
          type: "stat-card",
          title: "T",
          position: { x: 0, y: 0, w: 1, h: 1 },
          dataSource: { api: { type: "api", endpoint: "/x", method: "GET" } },
          display: { type: "stat-card", icon: "📊", iconColor: "red" },
          enabled: true,
        }}
        onClose={vi.fn()}
        onSelect={vi.fn()}
      />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("编辑 Widget");
  });

  it("handleConfirm + 校验通过 → 调用 onSelect", async () => {
    const onSelect = vi.fn();
    const { baseElement, getByText } = render(
      <WidgetSelector visible onClose={vi.fn()} onSelect={onSelect} />,
      { wrapper }
    );
    fireEvent.click(getByText("统计卡片"));
    await act(async () => {
      fireEvent.click(getByText("确 定"));
    });
    expect(onSelect).toHaveBeenCalled();
  });

  it("handleCancel → 调用 onClose", () => {
    const onClose = vi.fn();
    const { getByText } = render(<WidgetSelector visible onClose={onClose} onSelect={vi.fn()} />, {
      wrapper,
    });
    fireEvent.click(getByText("取 消"));
    expect(onClose).toHaveBeenCalled();
  });

  it("visible=true → DataSourceForm + DisplayConfigForm 渲染", () => {
    const { baseElement, getByText } = render(
      <WidgetSelector visible onClose={vi.fn()} onSelect={vi.fn()} />,
      { wrapper }
    );
    fireEvent.click(getByText("统计卡片"));
    expect(baseElement.querySelector('[data-testid="data-source-form"]')).toBeTruthy();
    expect(baseElement.querySelector('[data-testid="display-config-form"]')).toBeTruthy();
  });
});
