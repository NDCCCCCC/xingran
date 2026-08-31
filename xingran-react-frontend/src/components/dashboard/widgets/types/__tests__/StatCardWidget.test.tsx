/**
 * Phase 88 Batch232 — components/dashboard/widgets/types/StatCardWidget 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/hooks/useWidgetData", () => ({
  useWidgetData: vi.fn(() => ({
    data: { value: 42, label: "Total" },
    loading: false,
    error: null,
    refresh: vi.fn(),
  })),
}));

import { StatCardWidget } from "../StatCardWidget";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

const baseWidget: any = {
  id: "w1",
  type: "stat-card",
  title: "Default Title",
};

describe("dashboard/StatCardWidget", () => {
  it("渲染 value + label", () => {
    const display: any = { iconColor: "#000" };
    render(<StatCardWidget widget={baseWidget} display={display} />, { wrapper });
    expect(screen.getByText("42")).toBeInTheDocument();
    expect(screen.getByText("Total")).toBeInTheDocument();
  });

  it("loading=true → ReloadOutlined spin", async () => {
    const wd = await import("@/hooks/useWidgetData");
    vi.mocked(wd.useWidgetData).mockReturnValueOnce({
      data: null,
      loading: true,
      error: null,
      refresh: vi.fn(),
    } as any);
    const display: any = { iconColor: "#000" };
    const { container } = render(<StatCardWidget widget={baseWidget} display={display} />, {
      wrapper,
    });
    expect(container.querySelector(".anticon-reload")).toBeTruthy();
  });

  it("error → 显示错误", async () => {
    const wd = await import("@/hooks/useWidgetData");
    vi.mocked(wd.useWidgetData).mockReturnValueOnce({
      data: null,
      loading: false,
      error: "加载失败",
      refresh: vi.fn(),
    } as any);
    const display: any = { iconColor: "#000" };
    render(<StatCardWidget widget={baseWidget} display={display} />, { wrapper });
    expect(screen.getByText("加载失败")).toBeInTheDocument();
  });

  it("data=null → fallback value=label=widget.title", async () => {
    const wd = await import("@/hooks/useWidgetData");
    vi.mocked(wd.useWidgetData).mockReturnValueOnce({
      data: null,
      loading: false,
      error: null,
      refresh: vi.fn(),
    } as any);
    const display: any = { iconColor: "#000" };
    render(<StatCardWidget widget={baseWidget} display={display} />, { wrapper });
    expect(screen.getByText("Default Title")).toBeInTheDocument();
  });

  it("data 是字符串 → fallback value='-'", async () => {
    const wd = await import("@/hooks/useWidgetData");
    vi.mocked(wd.useWidgetData).mockReturnValueOnce({
      data: "not-an-object",
      loading: false,
      error: null,
      refresh: vi.fn(),
    } as any);
    const display: any = { iconColor: "#000" };
    render(<StatCardWidget widget={baseWidget} display={display} />, { wrapper });
    expect(screen.getByText("Default Title")).toBeInTheDocument();
  });

  it("data 含 totalDevices → 取 totalDevices", async () => {
    const wd = await import("@/hooks/useWidgetData");
    vi.mocked(wd.useWidgetData).mockReturnValueOnce({
      data: { totalDevices: 99, label: "Devices" },
      loading: false,
      error: null,
      refresh: vi.fn(),
    } as any);
    const display: any = { iconColor: "#000" };
    render(<StatCardWidget widget={baseWidget} display={display} />, { wrapper });
    expect(screen.getByText("99")).toBeInTheDocument();
  });

  it("display.prefix + suffix + percentage + decimals", async () => {
    const wd = await import("@/hooks/useWidgetData");
    vi.mocked(wd.useWidgetData).mockReturnValueOnce({
      data: { value: 12.345 },
      loading: false,
      error: null,
      refresh: vi.fn(),
    } as any);
    const display: any = { prefix: "$", suffix: "k", percentage: true, decimals: 2 };
    render(<StatCardWidget widget={baseWidget} display={display} />, { wrapper });
    // 12.345 → "$12.35k%"
    expect(screen.getByText(/\$12\.35k%/)).toBeInTheDocument();
  });

  it("trend up 渲染 ArrowUp", async () => {
    const wd = await import("@/hooks/useWidgetData");
    vi.mocked(wd.useWidgetData).mockReturnValueOnce({
      data: { value: 50, trend: { value: 10, direction: "up" } },
      loading: false,
      error: null,
      refresh: vi.fn(),
    } as any);
    const display: any = { iconColor: "#000" };
    const { container } = render(<StatCardWidget widget={baseWidget} display={display} />, {
      wrapper,
    });
    expect(container.querySelector(".anticon-arrow-up")).toBeTruthy();
    expect(screen.getByText("10%")).toBeInTheDocument();
  });

  it("trend down 渲染 ArrowDown", async () => {
    const wd = await import("@/hooks/useWidgetData");
    vi.mocked(wd.useWidgetData).mockReturnValueOnce({
      data: { value: 50, trend: { value: 5, direction: "down" } },
      loading: false,
      error: null,
      refresh: vi.fn(),
    } as any);
    const display: any = { iconColor: "#000" };
    const { container } = render(<StatCardWidget widget={baseWidget} display={display} />, {
      wrapper,
    });
    expect(container.querySelector(".anticon-arrow-down")).toBeTruthy();
  });

  it("data 为 0 → 显示 0", async () => {
    const wd = await import("@/hooks/useWidgetData");
    vi.mocked(wd.useWidgetData).mockReturnValueOnce({
      data: { value: 0 },
      loading: false,
      error: null,
      refresh: vi.fn(),
    } as any);
    const display: any = { iconColor: "#000" };
    render(<StatCardWidget widget={baseWidget} display={display} />, { wrapper });
    expect(screen.getByText("0")).toBeInTheDocument();
  });

  it("data 含 icon 渲染图标", async () => {
    const wd = await import("@/hooks/useWidgetData");
    vi.mocked(wd.useWidgetData).mockReturnValueOnce({
      data: { value: 100 },
      loading: false,
      error: null,
      refresh: vi.fn(),
    } as any);
    const display: any = { iconColor: "#00ff00", icon: <span>📊</span> };
    render(<StatCardWidget widget={baseWidget} display={display} />, { wrapper });
    expect(screen.getByText("📊")).toBeInTheDocument();
  });
});
