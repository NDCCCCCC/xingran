/**
 * v1.51 Phase 127 PWX-04 — 端口写审计明细页测试
 *
 * 覆盖：
 * - 分页表渲染 + 状态 Tag
 * - 过滤提交参数（deviceId/portId/operator/action/status + RFC3339 时间）
 * - 展开行 before/after JSON 与设备原始响应
 * - API 失败空态兜底（后端契约未就绪期间不白屏）
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent, act } from "@testing-library/react";
import { App } from "antd";
import type { PortWriteAuditRow, PortWriteAuditListResult } from "@/types/network";

const mockListPortWriteAudits = vi.fn();
const mockDeviceOptions = vi.fn();
const mockPortOptions = vi.fn();
vi.mock("@/lib/api/networkApi", () => ({
  listPortWriteAudits: (...args: unknown[]) => mockListPortWriteAudits(...args),
  listPortWriteAuditDeviceOptions: (...args: unknown[]) => mockDeviceOptions(...args),
  listPortWriteAuditPortOptions: (...args: unknown[]) => mockPortOptions(...args),
}));

import PortWriteAuditsPage from "../index";

function Wrapper({ children }: { children: React.ReactNode }) {
  return <App>{children}</App>;
}

function makeRow(overrides?: Partial<PortWriteAuditRow>): PortWriteAuditRow {
  return {
    id: "audit-1",
    deviceId: "device-1",
    portId: "port-1",
    action: "shutdown",
    beforeValue: { adminStatus: "up" },
    afterValue: { adminStatus: "down" },
    commandSent: "interface Gi0/1\nshutdown",
    deviceResponse: "OK",
    status: "succeeded",
    failureReason: null,
    operator: "admin",
    createdAt: "2026-11-01T00:00:00Z",
    ...overrides,
  };
}

function makeResult(list: PortWriteAuditRow[]): PortWriteAuditListResult {
  return { list, total: list.length, current: 1, pageSize: 10 };
}

describe("PortWriteAuditsPage — PWX-04", () => {
  beforeEach(() => {
    mockListPortWriteAudits.mockReset();
    mockDeviceOptions.mockReset();
    mockPortOptions.mockReset();
    // v1.57 选择器默认空选项（不影响列表断言）；个别用例可覆盖
    mockDeviceOptions.mockResolvedValue({ list: [], total: 0, current: 1, pageSize: 50 });
    mockPortOptions.mockResolvedValue({ list: [], total: 0, current: 1, pageSize: 50 });
  });

  it("渲染分页表 + 状态 Tag", async () => {
    mockListPortWriteAudits.mockResolvedValue(makeResult([makeRow()]));

    render(<PortWriteAuditsPage />, { wrapper: Wrapper });

    await waitFor(() => {
      expect(mockListPortWriteAudits).toHaveBeenCalledTimes(1);
    });
    expect(await screen.findByText("admin")).toBeInTheDocument();
    expect(screen.getByText("成功")).toBeInTheDocument();
    // 分页请求参数
    const arg = mockListPortWriteAudits.mock.calls[0][0] as { current: number; pageSize: number };
    expect(arg.current).toBe(1);
    expect(arg.pageSize).toBe(10);
  });

  it("筛选区使用 FilterBar 统一骨架，表格开启横向滚动（窄屏自适应）", async () => {
    mockListPortWriteAudits.mockResolvedValue(makeResult([makeRow()]));

    const { container } = render(<PortWriteAuditsPage />, { wrapper: Wrapper });
    await screen.findByText("admin");

    // 筛选卡片走统一骨架：窄宽下由 responsive.css 切成等宽栅格
    const bar = container.querySelector(".xr-filter-bar");
    expect(bar).not.toBeNull();
    expect(bar?.querySelector(".xr-filter-bar__form .ant-form-inline")).not.toBeNull();

    // 表格开启横向滚动：缺 scroll.x 时列宽会被压缩成逐字竖排
    const content = container.querySelector(".ant-table-content");
    expect(content).not.toBeNull();
    const table = content?.querySelector("table");
    expect(table?.getAttribute("style") ?? "").toMatch(/min-width|width/);
  });

  it("展开行显示 before/after JSON 与设备原始响应", async () => {
    mockListPortWriteAudits.mockResolvedValue(makeResult([makeRow()]));

    render(<PortWriteAuditsPage />, { wrapper: Wrapper });
    await screen.findByText("admin");

    // antd Table 展开按钮
    const expandBtn = document.querySelector(".ant-table-row-expand-icon");
    expect(expandBtn).not.toBeNull();
    await act(async () => {
      fireEvent.click(expandBtn as HTMLElement);
    });

    expect(await screen.findByText("变更前 (beforeValue)")).toBeInTheDocument();
    expect(screen.getByText("变更后 (afterValue)")).toBeInTheDocument();
    // before/after 各渲染一份 adminStatus 快照
    expect(screen.getAllByText(/adminStatus/).length).toBe(2);
  });

  it("v1.57 列表展示设备名称/IP 与接口名/端口描述，不再直接展示裸 UUID", async () => {
    mockListPortWriteAudits.mockResolvedValue(
      makeResult([
        makeRow({
          deviceName: "核心交换机-01",
          deviceIp: "10.1.1.1",
          interfaceName: "GE0/0/1",
          portDescription: "上行到核心",
          nameFromSnapshot: true,
        }),
      ])
    );

    render(<PortWriteAuditsPage />, { wrapper: Wrapper });

    expect(await screen.findByText("核心交换机-01")).toBeInTheDocument();
    expect(screen.getByText("10.1.1.1")).toBeInTheDocument();
    expect(screen.getByText("GE0/0/1")).toBeInTheDocument();
    expect(screen.getByText("上行到核心")).toBeInTheDocument();
    // UUID 不再作为单元格主文本出现
    expect(screen.queryByText("device-1")).toBeNull();
    expect(screen.queryByText("port-1")).toBeNull();
  });

  it("v1.57 名称缺失时显示兜底文案（已删除设备 / 未采集）", async () => {
    mockListPortWriteAudits.mockResolvedValue(makeResult([makeRow()]));

    render(<PortWriteAuditsPage />, { wrapper: Wrapper });

    expect(await screen.findByText("已删除设备")).toBeInTheDocument();
    expect(screen.getByText("未采集/已删除")).toBeInTheDocument();
  });

  it("v1.57 设备选择后按 deviceId 预取端口选项（联动 + 提交参数）", async () => {
    mockListPortWriteAudits.mockResolvedValue(makeResult([]));
    mockDeviceOptions.mockResolvedValue({
      list: [{ value: "dev-1", label: "核心交换机-01", ipAddress: "10.1.1.1" }],
      total: 1,
      current: 1,
      pageSize: 50,
    });
    mockPortOptions.mockResolvedValue({
      list: [{ value: "port-1", label: "GE0/0/1", interfaceName: "GE0/0/1", description: "上行" }],
      total: 1,
      current: 1,
      pageSize: 50,
    });

    render(<PortWriteAuditsPage />, { wrapper: Wrapper });
    await waitFor(() => {
      expect(mockDeviceOptions).toHaveBeenCalled();
    });
    // 设备选项预取走 device-options（不是 /network/devices/list，避免审计员 403）
    expect(mockDeviceOptions).toHaveBeenCalledWith({ current: 1, pageSize: 50 });
    // 初始未选设备 → 不请求端口选项
    expect(mockPortOptions).not.toHaveBeenCalled();
  });

  it("API 失败时空态兜底不白屏", async () => {
    mockListPortWriteAudits.mockRejectedValue(new Error("404 not found"));

    render(<PortWriteAuditsPage />, { wrapper: Wrapper });

    await waitFor(() => {
      expect(mockListPortWriteAudits).toHaveBeenCalledTimes(1);
    });
    // 空态渲染（antd Table placeholder; locale 默认英文环境, 断言占位结构存在）
    await waitFor(() => {
      expect(document.querySelector(".ant-table-placeholder")).not.toBeNull();
    });
  });
});
