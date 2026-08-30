/**
 * Phase 88 Batch133 — components/dashboard/layout/TemplatePreview 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent, waitFor } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/services/dashboardService", () => ({
  dashboardService: {
    getBatchWidgetData: vi.fn(() => Promise.resolve(new Map())),
  },
}));

vi.mock("../DashboardGrid", () => ({
  DashboardGrid: ({ widgets }: any) => (
    <div data-testid="dashboard-grid">
      {widgets?.map((w: any) => (
        <span key={w.id}>{w.title}</span>
      ))}
    </div>
  ),
}));

import TemplatePreview from "../TemplatePreview";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

const template = {
  id: "t1",
  name: "销售仪表盘",
  layout: {
    widgets: [
      {
        id: "w1",
        title: "指标卡片",
        type: "stat-card",
        position: { x: 0, y: 0, w: 1, h: 1 },
        dataSource: { api: { type: "api", endpoint: "/x", method: "GET" } },
        display: { type: "stat-card", icon: "📊", iconColor: "red" },
        enabled: true,
      },
      {
        id: "w2",
        title: "趋势图",
        type: "chart",
        position: { x: 1, y: 0, w: 1, h: 1 },
        dataSource: { api: { type: "api", endpoint: "/x", method: "GET" } },
        display: { type: "chart", chartType: "line", showLegend: true },
        enabled: true,
      },
    ],
  },
};

describe("TemplatePreview", () => {
  it("template=null → 返回 null", () => {
    const { baseElement } = render(
      <TemplatePreview visible template={null} onClose={vi.fn()} onApply={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.querySelector(".ant-modal")).toBeNull();
  });

  it("visible=true + template → 渲染 Modal + DashboardGrid", async () => {
    const { baseElement } = render(
      <TemplatePreview visible template={template as any} onClose={vi.fn()} onApply={vi.fn()} />,
      { wrapper }
    );
    await waitFor(() => {
      expect(baseElement.querySelector(".ant-modal")).toBeTruthy();
    });
    expect(baseElement.textContent).toContain("模板预览");
    expect(baseElement.textContent).toContain("销售仪表盘");
    expect(baseElement.querySelector('[data-testid="dashboard-grid"]')).toBeTruthy();
  });

  it("点击 使用此模板 → 调用 onApply(templateId, name+副本)", async () => {
    const onApply = vi.fn();
    const { baseElement, getByText } = render(
      <TemplatePreview visible template={template as any} onClose={vi.fn()} onApply={onApply} />,
      { wrapper }
    );
    await waitFor(() => {
      expect(baseElement.querySelector(".ant-modal")).toBeTruthy();
    });
    fireEvent.click(getByText("使用此模板"));
    expect(onApply).toHaveBeenCalledWith("t1", "销售仪表盘 - 副本");
  });

  it("点击 取消 → 调用 onClose", async () => {
    const onClose = vi.fn();
    const { baseElement, getByText } = render(
      <TemplatePreview visible template={template as any} onClose={onClose} onApply={vi.fn()} />,
      { wrapper }
    );
    await waitFor(() => {
      expect(baseElement.querySelector(".ant-modal")).toBeTruthy();
    });
    fireEvent.click(getByText("取消"));
    expect(onClose).toHaveBeenCalled();
  });

  it("visible=false → 重置状态不渲染 modal", () => {
    const { baseElement } = render(
      <TemplatePreview
        visible={false}
        template={template as any}
        onClose={vi.fn()}
        onApply={vi.fn()}
      />,
      { wrapper }
    );
    expect(baseElement.querySelector(".ant-modal-content")).toBeNull();
  });

  it("模板 widgets=[] → 不调 getBatchWidgetData", async () => {
    const { dashboardService } = await import("@/services/dashboardService");
    vi.mocked(dashboardService.getBatchWidgetData).mockClear();
    const emptyTpl = { ...template, layout: { widgets: [] } };
    const { baseElement } = render(
      <TemplatePreview visible template={emptyTpl as any} onClose={vi.fn()} onApply={vi.fn()} />,
      { wrapper }
    );
    await waitFor(() => {
      expect(baseElement.querySelector(".ant-modal")).toBeTruthy();
    });
    expect(dashboardService.getBatchWidgetData).not.toHaveBeenCalled();
  });

  it("加载失败 → 显示错误 Empty + 重新加载按钮", async () => {
    const { dashboardService } = await import("@/services/dashboardService");
    vi.mocked(dashboardService.getBatchWidgetData).mockRejectedValueOnce(new Error("网络错误"));
    const errSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const { baseElement, getByText } = render(
      <TemplatePreview visible template={template as any} onClose={vi.fn()} onApply={vi.fn()} />,
      { wrapper }
    );
    await waitFor(() => {
      expect(errSpy).toHaveBeenCalled();
    });
    expect(baseElement.textContent).toContain("重新加载");
    // Click reload to test handleReload
    vi.mocked(dashboardService.getBatchWidgetData).mockResolvedValueOnce(new Map());
    fireEvent.click(getByText("重新加载"));
    errSpy.mockRestore();
  });
});
