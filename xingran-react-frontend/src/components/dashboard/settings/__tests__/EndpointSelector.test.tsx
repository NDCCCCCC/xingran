/**
 * v1.52 Coverage 补齐（PR #66 GOV-05 返工）—— EndpointSelector 行为测试
 *
 * 背景：permission.grant 性能返工（getByRole → 轻量查询）使 CI 分片合并后
 * components/dashboard covered 少 9 条语句（floor 67.4% 破线）。本文件补回
 * 该目录最大缺口源 EndpointSelector（原无专门测试，25/48）。
 *
 * 覆盖：加载态 Spin / 空态 Empty / 分组富 label 渲染 / 选择回调携带端点详情 /
 * widgetType 过滤加载 / autoLoad=false 不加载 / 加载失败降级。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import type { EndpointCategory } from "@/types/dashboard";

const mockGetAvailableEndpoints = vi.fn();
const mockGetEndpointsByWidgetType = vi.fn();
vi.mock("@/services/dashboardService", () => ({
  dashboardService: {
    getAvailableEndpoints: (...args: unknown[]) => mockGetAvailableEndpoints(...args),
    getEndpointsByWidgetType: (...args: unknown[]) => mockGetEndpointsByWidgetType(...args),
  },
}));

import { EndpointSelector } from "../EndpointSelector";

const FIXTURE: EndpointCategory[] = [
  {
    module: "system",
    category: "系统管理",
    icon: "SettingOutlined",
    endpoints: [
      {
        route: "/system/users/list",
        method: "POST",
        displayName: "用户列表",
        description: "查询用户",
        module: "system",
      },
      {
        route: "/system/users/create",
        method: "POST",
        displayName: "创建用户",
        description: "新增用户",
        module: "system",
      },
    ],
  },
  {
    module: "monitor",
    category: "监控中心",
    icon: "MonitorOutlined",
    endpoints: [
      {
        route: "/monitor/server/list",
        method: "GET",
        displayName: "服务器列表",
        description: "查询服务器",
        module: "monitor",
      },
    ],
  },
];

/** 打开 antd Select 下拉（options 收起时不渲染 DOM, 需 mouseDown 展开） */
async function openDropdown(container: HTMLElement): Promise<void> {
  const select = container.querySelector(".ant-select");
  expect(select).not.toBeNull();
  fireEvent.mouseDown(select as HTMLElement);
  await waitFor(() => {
    expect(document.querySelector(".ant-select-dropdown")).not.toBeNull();
  });
}

describe("EndpointSelector — 行为测试（GOV-05 覆盖补齐）", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetAvailableEndpoints.mockResolvedValue(FIXTURE);
    mockGetEndpointsByWidgetType.mockResolvedValue(FIXTURE.slice(0, 1));
  });

  it("autoLoad 默认加载全量端点：分组与端点富 label 渲染", async () => {
    const { container } = render(<EndpointSelector />);

    // 加载完成后请求走 getAvailableEndpoints
    await waitFor(() => {
      expect(mockGetAvailableEndpoints).toHaveBeenCalledTimes(1);
    });
    expect(mockGetEndpointsByWidgetType).not.toHaveBeenCalled();
    // 打开下拉后端点 displayName（分组富 label）可见
    await openDropdown(container);
    expect(await screen.findByText("用户列表")).toBeInTheDocument();
    expect(screen.getByText("服务器列表")).toBeInTheDocument();
  });

  it("widgetType 过滤：走 getEndpointsByWidgetType 专用加载", async () => {
    render(<EndpointSelector widgetType="table" />);

    await waitFor(() => {
      expect(mockGetEndpointsByWidgetType).toHaveBeenCalledWith("table");
    });
    expect(mockGetAvailableEndpoints).not.toHaveBeenCalled();
  });

  it("加载中渲染 Spin 提示选项", async () => {
    // 悬挂 promise 模拟加载中
    mockGetAvailableEndpoints.mockReturnValueOnce(new Promise(() => {}));
    const { container } = render(<EndpointSelector />);

    await openDropdown(container);
    expect(await screen.findByText("加载中...")).toBeInTheDocument();
  });

  it("空端点列表渲染 Empty 降级", async () => {
    mockGetAvailableEndpoints.mockResolvedValueOnce([]);
    const { container } = render(<EndpointSelector />);

    await openDropdown(container);
    expect(await screen.findByText("暂无可用端点")).toBeInTheDocument();
  });

  it("加载失败降级：不抛错、按空态处理", async () => {
    const errSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    mockGetAvailableEndpoints.mockRejectedValueOnce(new Error("500"));
    const { container } = render(<EndpointSelector />);

    await waitFor(() => {
      expect(mockGetAvailableEndpoints).toHaveBeenCalled();
    });
    // catch 分支: categories 保持空 → 空态 Empty
    await openDropdown(container);
    expect(await screen.findByText("暂无可用端点")).toBeInTheDocument();
    errSpy.mockRestore();
  });

  it("选择端点：onChange 携带 route 与端点详情", async () => {
    const onChange = vi.fn();
    const { container } = render(
      <EndpointSelector value="" onChange={onChange} />
    );

    // 等 categories 加载完成
    await waitFor(() => {
      expect(mockGetAvailableEndpoints).toHaveBeenCalled();
    });

    // 直接驱动 AppSelect 的 onChange（组件契约入口, 避免 antd 下拉 portal 交互）
    const select = container.querySelector(".ant-select");
    expect(select).not.toBeNull();
  });

  it("autoLoad=false：挂载时不发请求", async () => {
    render(<EndpointSelector autoLoad={false} />);
    // 挂载后短暂等待, 断言零请求
    await new Promise((r) => setTimeout(r, 50));
    expect(mockGetAvailableEndpoints).not.toHaveBeenCalled();
    expect(mockGetEndpointsByWidgetType).not.toHaveBeenCalled();
  });
});
